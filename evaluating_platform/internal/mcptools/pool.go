package mcptools

import (
	"fmt"
	"sync"

	"evaluating_platform/internal/connector"
)

// ConnectorPool 线程安全的 Connector 池，按 assessment_id 索引
type ConnectorPool struct {
	mu   sync.RWMutex
	pool map[string]connector.TargetConnector
}

// NewConnectorPool 创建 ConnectorPool
func NewConnectorPool() *ConnectorPool {
	return &ConnectorPool{
		pool: make(map[string]connector.TargetConnector),
	}
}

// Register 注册 assessment_id 对应的 connector
func (p *ConnectorPool) Register(assessmentID string, conn connector.TargetConnector) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool[assessmentID] = conn
}

// Get 获取 assessment_id 对应的 connector
func (p *ConnectorPool) Get(assessmentID string) (connector.TargetConnector, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	conn, ok := p.pool[assessmentID]
	if !ok {
		return nil, fmt.Errorf("connector not found for assessment_id: %s", assessmentID)
	}
	return conn, nil
}

// Remove 删除 assessment_id 对应的 connector（评估结束后清理）
func (p *ConnectorPool) Remove(assessmentID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pool, assessmentID)
}
