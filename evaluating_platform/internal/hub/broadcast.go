package hub

import (
	"sync"
	"time"
)

// LogEvent 单次工具调用的实时日志事件
type LogEvent struct {
	AssessmentID string    `json:"assessment_id"`
	Iteration    int       `json:"iteration"`
	ToolName     string    `json:"tool_name"`
	Output       string    `json:"output"`
	Severity     string    `json:"severity,omitempty"`
	TokensUsed   int       `json:"tokens_used"`
	DurationMs   int64     `json:"duration_ms"`
	Timestamp    time.Time `json:"timestamp"`
}

// LogHub SSE 日志广播中心：assessment_id → subscriber channels
type LogHub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan LogEvent
}

// NewLogHub 创建广播中心
func NewLogHub() *LogHub {
	return &LogHub{
		subscribers: make(map[string][]chan LogEvent),
	}
}

// Subscribe 订阅指定评估任务的日志，返回只读 channel 和取消订阅函数
func (h *LogHub) Subscribe(assessmentID string) (<-chan LogEvent, func()) {
	ch := make(chan LogEvent, 64)

	h.mu.Lock()
	h.subscribers[assessmentID] = append(h.subscribers[assessmentID], ch)
	h.mu.Unlock()

	unsub := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subscribers[assessmentID]
		for i, c := range subs {
			if c == ch {
				h.subscribers[assessmentID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
		if len(h.subscribers[assessmentID]) == 0 {
			delete(h.subscribers, assessmentID)
		}
	}

	return ch, unsub
}

// Publish 向所有订阅该评估任务的 channel 广播事件（非阻塞）
func (h *LogHub) Publish(event LogEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subscribers[event.AssessmentID] {
		select {
		case ch <- event:
		default:
			// 丢弃：channel 已满，避免阻塞执行流
		}
	}
}
