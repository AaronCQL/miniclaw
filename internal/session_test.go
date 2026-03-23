package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"miniclaw/internal/models"
)

func newTestSessionStore(t *testing.T) *SessionStore {
	t.Helper()
	dir := t.TempDir()
	return NewSessionStore(filepath.Join(dir, "sessions.json"))
}

func TestSessionStore_GetSet(t *testing.T) {
	s := newTestSessionStore(t)

	s.SetIfAbsent(100, 0, "sess-abc")
	if got := s.Get(100, 0); got != "sess-abc" {
		t.Errorf("Get = %q, want %q", got, "sess-abc")
	}
}

func TestSessionStore_SetIfAbsent_NoOverwrite(t *testing.T) {
	s := newTestSessionStore(t)

	s.SetIfAbsent(100, 0, "first")
	s.SetIfAbsent(100, 0, "second")
	if got := s.Get(100, 0); got != "first" {
		t.Errorf("Get = %q, want %q (should not overwrite)", got, "first")
	}
}

func TestSessionStore_UpdateUsage(t *testing.T) {
	s := newTestSessionStore(t)
	s.SetIfAbsent(100, 0, "sess-1")

	usage := map[string]models.ModelUsage{
		"opus": {
			InputTokens:              100,
			CacheCreationInputTokens: 5000,
			CacheReadInputTokens:     2000,
			CostUSD:                  0.50,
			ContextWindow:            1000000,
		},
	}
	s.UpdateUsage(100, 0, usage, false)

	got := s.GetUsage(100, 0)
	if got.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-1")
	}
	if got.ContextTokens != 7100 {
		t.Errorf("ContextTokens = %d, want 7100", got.ContextTokens)
	}
	if got.ContextWindow != 1000000 {
		t.Errorf("ContextWindow = %d, want 1000000", got.ContextWindow)
	}
	if got.CostUSD != 0.50 {
		t.Errorf("CostUSD = %f, want 0.50", got.CostUSD)
	}
}

func TestSessionStore_CostAccumulates(t *testing.T) {
	s := newTestSessionStore(t)

	usage := map[string]models.ModelUsage{
		"opus": {InputTokens: 100, CostUSD: 0.10, ContextWindow: 1000000},
	}
	s.UpdateUsage(100, 0, usage, false)
	s.UpdateUsage(100, 0, usage, false)

	got := s.GetUsage(100, 0)
	if got.CostUSD != 0.20 {
		t.Errorf("CostUSD = %f, want 0.20 (should accumulate)", got.CostUSD)
	}
}

func TestSessionStore_ContextOverwrites(t *testing.T) {
	s := newTestSessionStore(t)

	s.UpdateUsage(100, 0, map[string]models.ModelUsage{
		"opus": {InputTokens: 1000, ContextWindow: 1000000},
	}, false)
	s.UpdateUsage(100, 0, map[string]models.ModelUsage{
		"opus": {InputTokens: 5000, ContextWindow: 1000000},
	}, false)

	got := s.GetUsage(100, 0)
	if got.ContextTokens != 5000 {
		t.Errorf("ContextTokens = %d, want 5000 (should overwrite, not accumulate)", got.ContextTokens)
	}
}

func TestSessionStore_GetUsage_Missing(t *testing.T) {
	s := newTestSessionStore(t)

	got := s.GetUsage(999, 0)
	if got.ContextWindow != 0 || got.CostUSD != 0 {
		t.Errorf("missing key should return zero value, got %+v", got)
	}
}

func TestSessionStore_MultipleModels(t *testing.T) {
	s := newTestSessionStore(t)

	usage := map[string]models.ModelUsage{
		"opus":   {InputTokens: 1000, CostUSD: 0.50, ContextWindow: 1000000},
		"sonnet": {InputTokens: 500, CostUSD: 0.10, ContextWindow: 200000},
	}
	s.UpdateUsage(100, 0, usage, false)

	got := s.GetUsage(100, 0)
	if got.ContextTokens != 1500 {
		t.Errorf("ContextTokens = %d, want 1500 (sum across models)", got.ContextTokens)
	}
	if got.CostUSD != 0.60 {
		t.Errorf("CostUSD = %f, want 0.60 (sum across models)", got.CostUSD)
	}
	if got.ContextWindow != 1000000 {
		t.Errorf("ContextWindow = %d, want 1000000 (largest)", got.ContextWindow)
	}
}

func TestSessionStore_CostOnly(t *testing.T) {
	s := newTestSessionStore(t)

	// First update with context
	s.UpdateUsage(100, 0, map[string]models.ModelUsage{
		"opus": {InputTokens: 50000, CostUSD: 0.50, ContextWindow: 1000000},
	}, false)

	// Second update as isolated session (costOnly=true)
	s.UpdateUsage(100, 0, map[string]models.ModelUsage{
		"opus": {InputTokens: 500, CostUSD: 0.05, ContextWindow: 1000000},
	}, true)

	got := s.GetUsage(100, 0)
	if got.ContextTokens != 50000 {
		t.Errorf("ContextTokens = %d, want 50000 (costOnly should not overwrite)", got.ContextTokens)
	}
	if got.CostUSD != 0.55 {
		t.Errorf("CostUSD = %f, want 0.55 (cost should still accumulate)", got.CostUSD)
	}
}

func TestSessionStore_TotalCost(t *testing.T) {
	s := newTestSessionStore(t)

	s.UpdateUsage(100, 0, map[string]models.ModelUsage{
		"opus": {CostUSD: 1.00},
	}, false)
	s.UpdateUsage(100, 200, map[string]models.ModelUsage{
		"opus": {CostUSD: 0.50},
	}, false)
	s.UpdateUsage(300, 0, map[string]models.ModelUsage{
		"opus": {CostUSD: 0.25},
	}, false)

	got := s.TotalCost()
	if got != 1.75 {
		t.Errorf("TotalCost = %f, want 1.75", got)
	}
}

func TestSessionStore_Clear(t *testing.T) {
	s := newTestSessionStore(t)

	s.SetIfAbsent(100, 200, "sess-1")
	s.UpdateUsage(100, 200, map[string]models.ModelUsage{
		"opus": {InputTokens: 50000, CostUSD: 1.50, ContextWindow: 1000000},
	}, false)

	s.Clear(100, 200)

	got := s.GetUsage(100, 200)
	if got.SessionID != "" {
		t.Errorf("SessionID = %q, want empty after clear", got.SessionID)
	}
	if got.ContextTokens != 0 || got.ContextWindow != 0 {
		t.Errorf("Context = %d/%d, want 0/0 after clear", got.ContextTokens, got.ContextWindow)
	}
	if got.LastCostUSD != 0 {
		t.Errorf("LastCostUSD = %f, want 0 after clear", got.LastCostUSD)
	}
	if got.CostUSD != 1.50 {
		t.Errorf("CostUSD = %f, want 1.50 (should be preserved)", got.CostUSD)
	}

	// Next SetIfAbsent should work since SessionID was cleared
	s.SetIfAbsent(100, 200, "sess-2")
	if got := s.Get(100, 200); got != "sess-2" {
		t.Errorf("Get after clear = %q, want %q", got, "sess-2")
	}
}

func TestSessionStore_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	// Write legacy format: map[string]string
	legacy := map[string]string{
		"100":     "sess-abc",
		"200:300": "sess-xyz",
	}
	data, _ := json.MarshalIndent(legacy, "", "  ")
	os.WriteFile(path, data, 0600)

	s := NewSessionStore(path)

	if got := s.Get(100, 0); got != "sess-abc" {
		t.Errorf("Get(100,0) = %q, want %q", got, "sess-abc")
	}
	if got := s.Get(200, 300); got != "sess-xyz" {
		t.Errorf("Get(200,300) = %q, want %q", got, "sess-xyz")
	}

	// Trigger a save by updating usage, then verify it reads back correctly
	s.UpdateUsage(100, 0, map[string]models.ModelUsage{
		"opus": {InputTokens: 500, CostUSD: 0.05, ContextWindow: 1000000},
	}, false)

	// Re-read from disk
	s2 := NewSessionStore(path)
	got := s2.GetUsage(100, 0)
	if got.SessionID != "sess-abc" {
		t.Errorf("SessionID after migration = %q, want %q", got.SessionID, "sess-abc")
	}
	if got.CostUSD != 0.05 {
		t.Errorf("CostUSD after migration = %f, want 0.05", got.CostUSD)
	}

	// Other entry should also survive the migration
	if got := s2.Get(200, 300); got != "sess-xyz" {
		t.Errorf("Get(200,300) after migration = %q, want %q", got, "sess-xyz")
	}
}
