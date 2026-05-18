package aihelper

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// AIHelperManager AI助手管理器，管理用户-会话-AIHelper的映射关系
type AIHelperManager struct {
	helpers map[string]map[string]*AIHelper // map[用户账号（唯一）]map[会话ID]*AIHelper
	mu      sync.RWMutex
}

// NewAIHelperManager 创建新的管理器实例
func NewAIHelperManager() *AIHelperManager {
	return &AIHelperManager{
		helpers: make(map[string]map[string]*AIHelper),
	}
}

// 获取或创建AIHelper
func (m *AIHelperManager) GetOrCreateAIHelper(ctx context.Context, userName string, sessionID string, opts CreateOptions) (*AIHelper, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if opts == nil {
		return nil, fmt.Errorf("create options is nil")
	}

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
			return helper, nil
		} else {
			newModel, err := factory.CreateAIModel(ctx, opts)
			if err != nil {
				return nil, err
			}
			helper.SwitchModel(newModel)
			return helper, nil
		}
	}

	// 创建新的AIHelper
	helper, err := factory.CreateAIHelper(ctx, sessionID, opts)
	if err != nil {
		return nil, err
	}

	userHelpers[sessionID] = helper
	return helper, nil
}

// 获取指定用户的指定会话的AIHelper
func (m *AIHelperManager) GetAIHelper(userName string, sessionID string) (*AIHelper, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return nil, false
	}

	helper, exists := userHelpers[sessionID]
	return helper, exists
}

// 移除指定用户的指定会话的AIHelper
func (m *AIHelperManager) RemoveAIHelper(userName string, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return
	}

	delete(userHelpers, sessionID)

	// 如果用户没有会话了，清理用户映射
	if len(userHelpers) == 0 {
		delete(m.helpers, userName)
	}
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
