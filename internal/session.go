package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type SessionStore struct {
	path     string
	sessions map[string]string // chatID (as string) → session ID
	mu       sync.RWMutex
}

func NewSessionStore(path string) (*SessionStore, error) {
	s := &SessionStore{
		path:     path,
		sessions: make(map[string]string),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading sessions file: %w", err)
	}

	if err := json.Unmarshal(data, &s.sessions); err != nil {
		return nil, fmt.Errorf("parsing sessions file: %w", err)
	}

	return s, nil
}

func (s *SessionStore) Get(chatID int64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[fmt.Sprintf("%d", chatID)]
}

func (s *SessionStore) Set(chatID int64, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[fmt.Sprintf("%d", chatID)] = sessionID
	s.save()
}

func (s *SessionStore) save() {
	data, err := json.MarshalIndent(s.sessions, "", "  ")
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
