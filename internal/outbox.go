package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// OutboxEntry represents a single file to send via Telegram.
type OutboxEntry struct {
	Path    string `json:"path"`
	Caption string `json:"caption,omitempty"`
}

// ReadOutbox reads and parses outbox.json, returning nil if it doesn't exist.
func ReadOutbox(path string) ([]OutboxEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading outbox: %w", err)
	}

	var entries []OutboxEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing outbox: %w", err)
	}

	return entries, nil
}

// RemoveOutbox deletes the outbox file. Errors are ignored.
func RemoveOutbox(path string) {
	os.Remove(path)
}

// ValidateOutboxEntry checks that the entry's path is within allowedDirs,
// exists, is a regular file, and is under 50MB.
func ValidateOutboxEntry(entry OutboxEntry, allowedDirs []string) error {
	absPath, err := filepath.Abs(entry.Path)
	if err != nil {
		return fmt.Errorf("invalid path %q", entry.Path)
	}

	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found", filepath.Base(absPath))
		}
		return fmt.Errorf("resolving path: %w", err)
	}

	if !isPathAllowed(resolved, allowedDirs) {
		return fmt.Errorf("path outside sandbox")
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("%s not found", filepath.Base(resolved))
	}

	if info.IsDir() {
		return fmt.Errorf("cannot send a directory")
	}

	const maxSize = 50 * 1024 * 1024
	if info.Size() > maxSize {
		return fmt.Errorf("%s exceeds 50MB limit (%d MB)", filepath.Base(resolved), info.Size()/(1024*1024))
	}

	return nil
}

func isPathAllowed(path string, allowedDirs []string) bool {
	for _, dir := range allowedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		prefix := filepath.Clean(absDir) + string(filepath.Separator)
		if strings.HasPrefix(path, prefix) || path == filepath.Clean(absDir) {
			return true
		}
	}
	return false
}
