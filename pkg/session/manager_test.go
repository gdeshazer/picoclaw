package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"telegram:123456", "telegram_123456"},
		{"discord:987654321", "discord_987654321"},
		{"slack:C01234", "slack_C01234"},
		{"no-colons-here", "no-colons-here"},
		{"multiple:colons:here", "multiple_colons_here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSave_WithColonInKey(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Create a session with a key containing colon (typical channel session key).
	key := "telegram:123456"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")

	// Save should succeed even though the key contains ':'
	if err := sm.Save(key); err != nil {
		t.Fatalf("Save(%q) failed: %v", key, err)
	}

	// The file on disk should use sanitized name.
	expectedFile := filepath.Join(tmpDir, "telegram_123456.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected session file %s to exist", expectedFile)
	}

	// Load into a fresh manager and verify the session round-trips.
	sm2 := NewSessionManager(tmpDir)
	history := sm2.GetHistory(key)
	if len(history) != 1 {
		t.Fatalf("expected 1 message after reload, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected message content %q, got %q", "hello", history[0].Content)
	}
}

func TestSave_RejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	badKeys := []string{"", ".", "..", "foo/bar", "foo\\bar"}
	for _, key := range badKeys {
		sm.GetOrCreate(key)
		if err := sm.Save(key); err == nil {
			t.Errorf("Save(%q) should have failed but didn't", key)
		}
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Create sessions in reverse order so Updated times differ.
	sm.GetOrCreate("session:a")
	sm.AddMessage("session:a", "user", "msg-a")

	sm.GetOrCreate("session:b")
	sm.AddMessage("session:b", "user", "msg-b")

	sessions := sm.List()
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	// Most recently updated session should be first (session:b was updated last).
	if sessions[0].Key != "session:b" {
		t.Errorf("expected first session to be 'session:b', got %q", sessions[0].Key)
	}
}

func TestListByPeer(t *testing.T) {
	sm := NewSessionManager("")

	sm.GetOrCreate("agent:main:telegram:direct:123456")
	sm.GetOrCreate("agent:main:telegram:direct:123456:named:project-a")
	sm.GetOrCreate("agent:main:telegram:direct:789000")

	results := sm.ListByPeer("123456")
	if len(results) != 2 {
		t.Fatalf("expected 2 sessions for peer 123456, got %d", len(results))
	}
	for _, s := range results {
		if !contains(s.Key, "123456") {
			t.Errorf("session key %q does not contain '123456'", s.Key)
		}
	}

	// Peer that matches only one session.
	results2 := sm.ListByPeer("789000")
	if len(results2) != 1 {
		t.Fatalf("expected 1 session for peer 789000, got %d", len(results2))
	}

	// Non-existent peer returns empty.
	results3 := sm.ListByPeer("nonexistent")
	if len(results3) != 0 {
		t.Errorf("expected 0 sessions for nonexistent peer, got %d", len(results3))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDelete_RemovesSessionAndFile(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	key := "agent:main:telegram:direct:555"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")
	if err := sm.Save(key); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := sm.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Session should be gone from memory.
	history := sm.GetHistory(key)
	if len(history) != 0 {
		t.Errorf("expected empty history after delete, got %d messages", len(history))
	}

	// File should be gone from disk.
	sessions := sm.List()
	for _, s := range sessions {
		if s.Key == key {
			t.Errorf("session %q still present in List() after delete", key)
		}
	}
}

func TestDelete_NonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Deleting a key that was never created should not error.
	if err := sm.Delete("nonexistent:key"); err != nil {
		t.Errorf("Delete of nonexistent key returned error: %v", err)
	}
}

func TestDelete_NoStorage(t *testing.T) {
	sm := NewSessionManager("")

	sm.GetOrCreate("key:a")
	if err := sm.Delete("key:a"); err != nil {
		t.Errorf("Delete with no storage returned error: %v", err)
	}
}
