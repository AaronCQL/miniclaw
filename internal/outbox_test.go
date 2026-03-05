package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeOutbox(t *testing.T, path string, entries []OutboxEntry) {
	t.Helper()
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal outbox: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write outbox: %v", err)
	}
}

func TestReadOutbox_Valid(t *testing.T) {
	dir := t.TempDir()
	outbox := filepath.Join(dir, "outbox.json")

	want := []OutboxEntry{
		{Path: "/tmp/file.txt", Caption: "a caption"},
		{Path: "/tmp/photo.png"},
	}
	writeOutbox(t, outbox, want)

	got, err := ReadOutbox(outbox)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].Path != want[0].Path || got[0].Caption != want[0].Caption {
		t.Errorf("entry 0: got %+v, want %+v", got[0], want[0])
	}
	if got[1].Path != want[1].Path || got[1].Caption != "" {
		t.Errorf("entry 1: got %+v, want %+v", got[1], want[1])
	}
}

func TestReadOutbox_Missing(t *testing.T) {
	entries, err := ReadOutbox("/nonexistent/outbox.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Fatalf("expected nil, got %v", entries)
	}
}

func TestReadOutbox_EmptyArray(t *testing.T) {
	dir := t.TempDir()
	outbox := filepath.Join(dir, "outbox.json")
	os.WriteFile(outbox, []byte("[]"), 0644)

	got, err := ReadOutbox(outbox)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(got))
	}
}

func TestReadOutbox_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	outbox := filepath.Join(dir, "outbox.json")
	os.WriteFile(outbox, []byte("{bad json"), 0644)

	_, err := ReadOutbox(outbox)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestRemoveOutbox(t *testing.T) {
	dir := t.TempDir()
	outbox := filepath.Join(dir, "outbox.json")
	os.WriteFile(outbox, []byte("[]"), 0644)

	RemoveOutbox(outbox)

	if _, err := os.Stat(outbox); !os.IsNotExist(err) {
		t.Fatal("outbox file should have been deleted")
	}
}

func TestRemoveOutbox_Missing(t *testing.T) {
	// Should not panic or error.
	RemoveOutbox("/nonexistent/outbox.json")
}

func TestValidateOutboxEntry_Valid(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	err := ValidateOutboxEntry(OutboxEntry{Path: f}, []string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOutboxEntry_OutsideSandbox(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	f := filepath.Join(other, "secret.txt")
	os.WriteFile(f, []byte("secret"), 0644)

	err := ValidateOutboxEntry(OutboxEntry{Path: f}, []string{dir})
	if err == nil {
		t.Fatal("expected sandbox error")
	}
}

func TestValidateOutboxEntry_NotFound(t *testing.T) {
	dir := t.TempDir()
	err := ValidateOutboxEntry(OutboxEntry{Path: filepath.Join(dir, "missing.txt")}, []string{dir})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestValidateOutboxEntry_Directory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	os.MkdirAll(sub, 0755)

	err := ValidateOutboxEntry(OutboxEntry{Path: sub}, []string{dir})
	if err == nil {
		t.Fatal("expected directory error")
	}
}

func TestValidateOutboxEntry_SymlinkTraversal(t *testing.T) {
	allowed := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	os.WriteFile(secret, []byte("secret"), 0644)

	link := filepath.Join(allowed, "link.txt")
	os.Symlink(secret, link)

	err := ValidateOutboxEntry(OutboxEntry{Path: link}, []string{allowed})
	if err == nil {
		t.Fatal("expected sandbox error for symlink traversal")
	}
}

func TestIsPathAllowed(t *testing.T) {
	tests := []struct {
		path    string
		dirs    []string
		allowed bool
	}{
		{"/home/user/workspace/file.txt", []string{"/home/user/workspace"}, true},
		{"/home/user/workspace/sub/file.txt", []string{"/home/user/workspace"}, true},
		{"/etc/passwd", []string{"/home/user/workspace"}, false},
		{"/home/user/workspacefile.txt", []string{"/home/user/workspace"}, false},
		{"/home/user/file.txt", []string{"/home/user/workspace", "/home/user"}, true},
	}

	for _, tt := range tests {
		got := isPathAllowed(tt.path, tt.dirs)
		if got != tt.allowed {
			t.Errorf("isPathAllowed(%q, %v) = %v, want %v", tt.path, tt.dirs, got, tt.allowed)
		}
	}
}
