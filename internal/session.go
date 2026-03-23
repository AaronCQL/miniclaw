package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"miniclaw/internal/models"
)

type SessionData struct {
	SessionID     string  `json:"sessionID"`
	ContextTokens int     `json:"contextTokens,omitempty"`
	ContextWindow int     `json:"contextWindow,omitempty"`
	CostUSD       float64 `json:"costUSD,omitempty"`
	LastCostUSD   float64 `json:"lastCostUSD,omitempty"`
}

// All reads and writes go directly to disk; the mutex serialises Go-side access only.
type SessionStore struct {
	path string
	mu   sync.Mutex
}

func NewSessionStore(path string) *SessionStore {
	return &SessionStore{path: path}
}

// sessionKey returns the map key for a given chat/thread pair.
// threadID == 0 uses plain "chatID" for backward compatibility with existing sessions.
// threadID > 0 uses "chatID:threadID".
func sessionKey(chatID, threadID int64) string {
	if threadID == 0 {
		return fmt.Sprintf("%d", chatID)
	}
	return fmt.Sprintf("%d:%d", chatID, threadID)
}

func (s *SessionStore) Get(chatID, threadID int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions := s.load()
	return sessions[sessionKey(chatID, threadID)].SessionID
}

// SetIfAbsent writes the session ID only if no session exists for this key yet.
// This respects sessions written by the agent (e.g. /migrate) or by the user
// editing sessions.json directly.
func (s *SessionStore) SetIfAbsent(chatID, threadID int64, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions := s.load()
	key := sessionKey(chatID, threadID)
	if sessions[key].SessionID != "" {
		return
	}
	entry := sessions[key]
	entry.SessionID = sessionID
	sessions[key] = entry
	s.save(sessions)
}

// costOnly skips the context snapshot - used for isolated sessions whose context is throwaway.
func (s *SessionStore) UpdateUsage(chatID, threadID int64, modelUsage map[string]models.ModelUsage, costOnly bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions := s.load()
	key := sessionKey(chatID, threadID)
	entry := sessions[key]

	var contextTokens, contextWindow int
	var cost float64
	for _, m := range modelUsage {
		contextTokens += m.InputTokens + m.CacheCreationInputTokens + m.CacheReadInputTokens
		cost += m.CostUSD
		if m.ContextWindow > contextWindow {
			contextWindow = m.ContextWindow
		}
	}

	if !costOnly {
		entry.ContextTokens = contextTokens
		entry.ContextWindow = contextWindow
	}
	entry.LastCostUSD = cost
	entry.CostUSD += cost

	sessions[key] = entry
	s.save(sessions)
}

func (s *SessionStore) GetUsage(chatID, threadID int64) SessionData {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions := s.load()
	return sessions[sessionKey(chatID, threadID)]
}

func (s *SessionStore) TotalCost() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var total float64
	for _, entry := range s.load() {
		total += entry.CostUSD
	}
	return total
}

// Supports legacy map[string]string format for backward compatibility.
func (s *SessionStore) load() map[string]SessionData {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("error reading sessions file: %v", err)
		}
		return make(map[string]SessionData)
	}

	sessions := make(map[string]SessionData)
	if err := json.Unmarshal(data, &sessions); err == nil {
		return sessions
	}

	legacy := make(map[string]string)
	if err := json.Unmarshal(data, &legacy); err != nil {
		log.Printf("error parsing sessions file: %v", err)
		return make(map[string]SessionData)
	}

	for k, v := range legacy {
		sessions[k] = SessionData{SessionID: v}
	}
	return sessions
}

func (s *SessionStore) save(sessions map[string]SessionData) {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		log.Printf("error marshaling sessions: %v", err)
		return
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, "sessions-*.json")
	if err != nil {
		log.Printf("error creating temp file for sessions: %v", err)
		return
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		log.Printf("error writing sessions temp file: %v", err)
		return
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		log.Printf("error closing sessions temp file: %v", err)
		return
	}

	if err := os.Chmod(tmp.Name(), 0600); err != nil {
		os.Remove(tmp.Name())
		log.Printf("error setting sessions file permissions: %v", err)
		return
	}

	if err := os.Rename(tmp.Name(), s.path); err != nil {
		os.Remove(tmp.Name())
		log.Printf("error renaming sessions file: %v", err)
	}
}
