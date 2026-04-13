package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettings_DefaultsWhenFileMissing(t *testing.T) {
	dir := t.TempDir()

	s := LoadSettings(dir)

	if s.StatusLevel != StatusText {
		t.Fatalf("StatusLevel = %q, want %q", s.StatusLevel, StatusText)
	}
	if s.Effort != EffortDefault {
		t.Fatalf("Effort = %q, want %q", s.Effort, EffortDefault)
	}
}

func TestLoadSettings_DefaultsWhenFileInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{invalid"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	s := LoadSettings(dir)

	if s.StatusLevel != StatusText {
		t.Fatalf("StatusLevel = %q, want %q", s.StatusLevel, StatusText)
	}
	if s.Effort != EffortDefault {
		t.Fatalf("Effort = %q, want %q", s.Effort, EffortDefault)
	}
}

func TestLoadSettings_NormalizesEmptyEffort(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"statusLevel":"verbose"}`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	s := LoadSettings(dir)

	if s.StatusLevel != StatusVerbose {
		t.Fatalf("StatusLevel = %q, want %q", s.StatusLevel, StatusVerbose)
	}
	if s.Effort != EffortDefault {
		t.Fatalf("Effort = %q, want %q", s.Effort, EffortDefault)
	}
}
