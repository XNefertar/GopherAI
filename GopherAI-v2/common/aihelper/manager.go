package aihelper

import (
	"context"
	"fmt"
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

// CountUserSessions 返回某个用户当前在内存中的会话数量，主要用于压测时校验造数是否成功。
func (m *AIHelperManager) CountUserSessions(userName string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userHelpers, exists := m.helpers[userName]
	if !exists {
		return 0
	}

	return len(userHelpers)
}

// SeedUserSessions 仅供压测使用：在内存中为指定用户预生成 count 个会话。
// 每次调用都会重置该用户的会话映射，避免复用上一次压测的残留数据。
func (m *AIHelperManager) SeedUserSessions(userName string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userHelpers := make(map[string]*AIHelper, count)
	m.helpers[userName] = userHelpers

	for i := 0; i < count; i++ {
		sessionID := fmt.Sprintf("bench-session-%06d", i)
		userHelpers[sessionID] = NewBenchmarkAIHelper(sessionID)
	}
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
