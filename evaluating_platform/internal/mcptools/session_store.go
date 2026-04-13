package mcptools

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// PayloadItem represents a single payload in the pipeline.
type PayloadItem struct {
	Index           int    `json:"index"`
	OriginalContent string `json:"original_content"` // ????????????
	CombinedText    string `json:"combined_text"`    // ??+????????
	EnhancedText    string `json:"enhanced_text"`    // ??????
	Language        string `json:"language"`         // ?????multilingual ???
	Sensitive       bool   `json:"sensitive,omitempty"`
	QuestionSummary string `json:"question_summary,omitempty"`
	PayloadSummary  string `json:"payload_summary,omitempty"`
}

// ExecutionResult represents the result of executing a single payload.
type ExecutionResult struct {
	Index           int    `json:"index"`
	OriginalContent string `json:"original_content"`
	EnhancedContent string `json:"enhanced_content"`
	QuestionSummary string `json:"question_summary,omitempty"`
	PayloadSummary  string `json:"payload_summary,omitempty"`
	Sensitive       bool   `json:"sensitive,omitempty"`
	TargetResponse  string `json:"target_response"`
	AttackSuccess   bool   `json:"attack_success"`
	AttackReason    string `json:"attack_reason"`
	Error           string `json:"error"`
}

// UnmarshalJSON keeps backward compatibility with older logs that used PascalCase keys.
func (r *ExecutionResult) UnmarshalJSON(data []byte) error {
	type alias ExecutionResult
	aux := struct {
		alias
		LegacyIndex           *int   `json:"Index"`
		LegacyOriginalContent string `json:"OriginalContent"`
		LegacyEnhancedContent string `json:"EnhancedContent"`
		LegacyTargetResponse  string `json:"TargetResponse"`
		LegacyAttackSuccess   *bool  `json:"AttackSuccess"`
		LegacyAttackReason    string `json:"AttackReason"`
		LegacyError           string `json:"Error"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*r = ExecutionResult(aux.alias)
	if aux.LegacyIndex != nil && r.Index == 0 {
		r.Index = *aux.LegacyIndex
	}
	if r.OriginalContent == "" {
		r.OriginalContent = aux.LegacyOriginalContent
	}
	if r.EnhancedContent == "" {
		r.EnhancedContent = aux.LegacyEnhancedContent
	}
	if r.TargetResponse == "" {
		r.TargetResponse = aux.LegacyTargetResponse
	}
	if aux.LegacyAttackSuccess != nil && !r.AttackSuccess {
		r.AttackSuccess = *aux.LegacyAttackSuccess
	}
	if r.AttackReason == "" {
		r.AttackReason = aux.LegacyAttackReason
	}
	if r.Error == "" {
		r.Error = aux.LegacyError
	}
	return nil
}

// SessionData holds the intermediate data for a pipeline session.
type SessionData struct {
	Payloads   []PayloadItem
	Results    []ExecutionResult
	LastAccess time.Time
}

// SessionStore is an in-memory store for pipeline session data, keyed by session ID.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
	ttl      time.Duration
}

// NewSessionStore creates a new SessionStore with the given TTL and starts a
// background goroutine that periodically removes expired sessions.
func NewSessionStore(ttl time.Duration) *SessionStore {
	s := &SessionStore{
		sessions: make(map[string]*SessionData),
		ttl:      ttl,
	}
	go s.cleanup()
	return s
}

// SetPayloads stores the payload list for the given session ID.
func (s *SessionStore) SetPayloads(sessionID string, payloads []PayloadItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sd, ok := s.sessions[sessionID]
	if !ok {
		sd = &SessionData{}
		s.sessions[sessionID] = sd
	}
	sd.Payloads = payloads
	sd.LastAccess = time.Now()
}

// GetPayloads returns the payload list for the given session ID.
// Returns an error if the session does not exist.
func (s *SessionStore) GetPayloads(sessionID string) ([]PayloadItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sd, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	sd.LastAccess = time.Now()
	return sd.Payloads, nil
}

// SetResults stores the execution results for the given session ID.
func (s *SessionStore) SetResults(sessionID string, results []ExecutionResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sd, ok := s.sessions[sessionID]
	if !ok {
		sd = &SessionData{}
		s.sessions[sessionID] = sd
	}
	sd.Results = results
	sd.LastAccess = time.Now()
}

// GetResults returns the execution results for the given session ID.
// Returns an error if the session does not exist.
func (s *SessionStore) GetResults(sessionID string) ([]ExecutionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sd, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	sd.LastAccess = time.Now()
	return sd.Results, nil
}

// cleanup runs in a background goroutine, removing expired sessions every minute.
func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, sd := range s.sessions {
			if now.Sub(sd.LastAccess) > s.ttl {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}
