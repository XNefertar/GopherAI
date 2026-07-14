package aihelper

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	appconfig "GopherAI/config"
)

// AIHelperManager AI助手管理器，管理用户-会话-AIHelper的映射关系。
//
// Phase 2 起引入 LRU + 空闲 TTL 容量治理：
//   - 用 lru 双向链表维护「最近使用」顺序，超过 maxSessions 时淘汰最久未使用会话；
//   - 后台 sweeper 按 idleTimeout 回收长时间空闲的会话；
//   - 淘汰前置：先 Flush 把未落库消息写回 DB，避免上下文丢失。
type AIHelperManager struct {
	helpers    map[string]map[string]*AIHelper // map[用户账号（唯一）]map[会话ID]*AIHelper
	mu         sync.RWMutex
	lru        *list.List // front=MRU，back=LRU；element.Value = *AIHelper
	maxSessions int
	idleTimeout time.Duration
	stopCh     chan struct{}
}

// NewAIHelperManager 创建新的管理器实例。
// 容量与空闲阈值来自 TOML（sessionCacheConfig），缺省时回退到保守默认值。
func NewAIHelperManager() *AIHelperManager {
	cfg := appconfig.GetConfig()
	maxSessions := cfg.SessionCache.MaxSessions
	if maxSessions <= 0 {
		maxSessions = 10000
	}
	idle := time.Duration(cfg.SessionCache.IdleTimeoutSec) * time.Second
	if idle <= 0 {
		idle = 30 * time.Minute
	}
	return &AIHelperManager{
		helpers:     make(map[string]map[string]*AIHelper),
		lru:         list.New(),
		maxSessions: maxSessions,
		idleTimeout: idle,
		stopCh:      make(chan struct{}),
	}
}

// 获取或创建AIHelper
func (m *AIHelperManager) GetOrCreateAIHelper(ctx context.Context, userName string, sessionID string, opts CreateOptions) (*AIHelper, error) {
	if opts == nil {
		return nil, fmt.Errorf("create options is nil")
	}

	m.mu.Lock()
	// 获取用户的会话映射
	userHelpers, exists := m.helpers[userName]
	if !exists {
		userHelpers = make(map[string]*AIHelper)
		m.helpers[userName] = userHelpers
	}

	// 检查会话是否已存在
	helper, exists := userHelpers[sessionID]
	factory := GetGlobalFactory()
	if exists {
		if helper.GetModelType() == opts.ModelType() {
			m.touchLocked(helper)
			m.mu.Unlock()
			return helper, nil
		}
		m.mu.Unlock()
		newModel, err := factory.CreateAIModel(ctx, opts)
		if err != nil {
			return nil, err
		}
		helper.SwitchModel(newModel)
		m.mu.Lock()
		m.touchLocked(helper)
		m.mu.Unlock()
		return helper, nil
	}

	// 创建新的AIHelper
	helper, err := factory.CreateAIHelper(ctx, sessionID, opts)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	helper.UserName = userName
	userHelpers[sessionID] = helper
	m.pushFrontLocked(helper)

	// 容量回收：超过上限则淘汰最久未使用的会话。
	// 先摘离（脱离 map+lru，避免并发访问）再在锁外 Flush，避免持锁写 DB 阻塞其他会话。
	var victims []*AIHelper
	for m.lru.Len() > m.maxSessions {
		back := m.lru.Back()
		if back == nil {
			break
		}
		victim := back.Value.(*AIHelper)
		m.detachLocked(victim)
		victims = append(victims, victim)
	}
	m.mu.Unlock()

	for _, v := range victims {
		if err := v.Flush(ctx); err != nil {
			log.Printf("[aihelper] evict flush session=%s failed: %v", v.SessionID, err)
		}
	}

	// 锁外惰性加载历史：避免 DB 查询阻塞其他会话的并发创建。
	// Hydrate 内部有 hydrated 标记保证幂等，即使并发重复创建也只加载一次。
	if err := helper.Hydrate(ctx); err != nil {
		log.Printf("[aihelper] hydrate session=%s failed: %v", sessionID, err)
	}
	helper.Touch()
	return helper, nil
}

// 获取指定用户的指定会话的AIHelper
func (m *AIHelperManager) GetAIHelper(userName string, sessionID string) (*AIHelper, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return nil, false
	}

	helper, exists := userHelpers[sessionID]
	if !exists {
		return nil, false
	}
	m.touchLocked(helper)
	return helper, true
}

// 移除指定用户的指定会话的AIHelper
func (m *AIHelperManager) RemoveAIHelper(userName string, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return
	}

	helper, exists := userHelpers[sessionID]
	if !exists {
		return
	}
	m.detachLocked(helper)
}

// 获取指定用户的所有会话ID
func (m *AIHelperManager) GetUserSessions(userName string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return []string{}
	}

	sessionIDs := make([]string, 0, len(userHelpers))
	//取出所有的key
	for sessionID := range userHelpers {
		sessionIDs = append(sessionIDs, sessionID)
	}

	return sessionIDs
}

// 全局管理器实例
var globalManager *AIHelperManager
var once sync.Once

// GetGlobalManager 获取全局管理器实例
func GetGlobalManager() *AIHelperManager {
	once.Do(func() {
		globalManager = NewAIHelperManager()
	})
	return globalManager
}

// GetOrCreateAIHelperWithAutoRoute 在 modelType=auto 场景下，
// 通过全局混合路由器先决策再走 GetOrCreateAIHelper 的统一入口。
//
// 该方法的好处：
//  1. service 层无需关心“具体走哪个模型”，只需要把 auto 透传过来；
//  2. 路由决策与创建/切换 helper 是一体的，便于后续统一埋点（命中率、升级率、成本）。
//
// 返回值除了 helper 之外，还会返回一次决策结果，便于上层做日志或链路追踪。
func (m *AIHelperManager) GetOrCreateAIHelperWithAutoRoute(
	ctx context.Context,
	userName string,
	sessionID string,
	question string,
	stream bool,
) (*AIHelper, RouteDecision, error) {
	router := GetGlobalRouter()
	decision, err := router.Route(ctx, userName, sessionID, question, stream)
	if err != nil {
		return nil, RouteDecision{}, fmt.Errorf("hybrid router route failed: %w", err)
	}
	log.Printf("[router] user=%s session=%s reason=%s model=%s",
		userName, sessionID, decision.Reason, decision.ModelType)

	helper, err := m.GetOrCreateAIHelper(ctx, userName, sessionID, decision.Options)
	if err != nil {
		return nil, decision, err
	}
	return helper, decision, nil
}

// ---- Phase 2 容量治理内部方法 ----

// touchLocked 在已持有 m.mu 时刷新访问时间并把节点移到 LRU 队首。
func (m *AIHelperManager) touchLocked(h *AIHelper) {
	h.Touch()
	if h.lruElem != nil {
		m.lru.MoveToFront(h.lruElem)
	}
}

// pushFrontLocked 在已持有 m.mu 时把 helper 加入 LRU 队首。
func (m *AIHelperManager) pushFrontLocked(h *AIHelper) {
	h.Touch()
	h.lruElem = m.lru.PushFront(h)
}

// detachLocked 在已持有 m.mu 时把 helper 从 LRU 与两级 map 中摘离（摘离后不再被并发访问）。
func (m *AIHelperManager) detachLocked(h *AIHelper) {
	if h.lruElem != nil {
		m.lru.Remove(h.lruElem)
		h.lruElem = nil
	}
	if uh, ok := m.helpers[h.UserName]; ok {
		delete(uh, h.SessionID)
		if len(uh) == 0 {
			delete(m.helpers, h.UserName)
		}
	}
}

// MarkPersisted 由 MQ 消费者落库成功后回灌，标记该会话最早一条未落库消息为已落库。
// 这样淘汰前的 Flush 只会写真正未落库的消息，避免与 MQ 消费者重复落库。
func (m *AIHelperManager) MarkPersisted(userName, sessionID string) {
	m.mu.RLock()
	uh, ok := m.helpers[userName]
	if !ok {
		m.mu.RUnlock()
		return
	}
	h, ok := uh[sessionID]
	m.mu.RUnlock()
	if ok {
		h.MarkPersisted()
	}
}

// Start 启动后台空闲回收 sweeper。
func (m *AIHelperManager) Start() {
	go m.sweep()
}

// Stop 停止后台 sweeper。
func (m *AIHelperManager) Stop() {
	select {
	case <-m.stopCh:
		// 已关闭，避免重复 close panic
	default:
		close(m.stopCh)
	}
}

func (m *AIHelperManager) sweep() {
	ticker := time.NewTicker(m.idleTimeout / 2)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.evictIdle()
		}
	}
}

// evictIdle 回收空闲超过 idleTimeout 的会话（先摘离再锁外 Flush）。
//
// 优化点（空间换时间的零成本版）：LRU 链表严格按 lastAccess 降序
// （front=最近访问, back=最久未访问，见 touchLocked/pushFrontLocked），
// 因此从 back 向前扫描时，一旦遇到尚未过期的节点，其前方所有节点必然
// 更未过期，可直接 break 终止，无需继续遍历整条链表——将平均复杂度
// 从 O(n) 降为 O(k)（k=实际过期会话数）。
func (m *AIHelperManager) evictIdle() {
	now := time.Now()
	var victims []*AIHelper
	m.mu.Lock()
	for e := m.lru.Back(); e != nil; {
		h := e.Value.(*AIHelper)
		// LRU 严格按 lastAccess 降序：back 最旧。
		// 一旦遇到未过期的，更前面的必然也未过期，直接终止扫描。
		if h.idleFor(now) <= m.idleTimeout {
			break
		}
		next := e.Prev()
		m.detachLocked(h)
		victims = append(victims, h)
		e = next
	}
	m.mu.Unlock()

	for _, h := range victims {
		if err := h.Flush(context.Background()); err != nil {
			log.Printf("[aihelper] idle evict flush session=%s failed: %v", h.SessionID, err)
		}
	}
}

// FlushAll 停机兜底：把当前所有内存会话中未落库的消息直接写回 DB，避免停机丢上下文。
func (m *AIHelperManager) FlushAll(ctx context.Context) {
	var all []*AIHelper
	m.mu.RLock()
	for _, uh := range m.helpers {
		for _, h := range uh {
			all = append(all, h)
		}
	}
	m.mu.RUnlock()

	for _, h := range all {
		if err := h.Flush(ctx); err != nil {
			log.Printf("[aihelper] flushall session=%s failed: %v", h.SessionID, err)
		}
	}
}
