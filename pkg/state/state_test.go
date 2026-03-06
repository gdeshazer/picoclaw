package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAtomicSave(t *testing.T) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	// Test SetLastChannel
	err = sm.SetLastChannel("test-channel")
	if err != nil {
		t.Fatalf("SetLastChannel failed: %v", err)
	}

	// Verify the channel was saved
	lastChannel := sm.GetLastChannel()
	if lastChannel != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", lastChannel)
	}

	// Verify timestamp was updated
	if sm.GetTimestamp().IsZero() {
		t.Error("Expected timestamp to be updated")
	}

	// Verify state file exists
	stateFile := filepath.Join(tmpDir, "state", "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("Expected state file to exist")
	}

	// Create a new manager to verify persistence
	sm2 := NewManager(tmpDir)
	if sm2.GetLastChannel() != "test-channel" {
		t.Errorf("Expected persistent channel 'test-channel', got '%s'", sm2.GetLastChannel())
	}
}

func TestSetLastChatID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	// Test SetLastChatID
	err = sm.SetLastChatID("test-chat-id")
	if err != nil {
		t.Fatalf("SetLastChatID failed: %v", err)
	}

	// Verify the chat ID was saved
	lastChatID := sm.GetLastChatID()
	if lastChatID != "test-chat-id" {
		t.Errorf("Expected chat ID 'test-chat-id', got '%s'", lastChatID)
	}

	// Verify timestamp was updated
	if sm.GetTimestamp().IsZero() {
		t.Error("Expected timestamp to be updated")
	}

	// Create a new manager to verify persistence
	sm2 := NewManager(tmpDir)
	if sm2.GetLastChatID() != "test-chat-id" {
		t.Errorf("Expected persistent chat ID 'test-chat-id', got '%s'", sm2.GetLastChatID())
	}
}

func TestAtomicity_NoCorruptionOnInterrupt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	// Write initial state
	err = sm.SetLastChannel("initial-channel")
	if err != nil {
		t.Fatalf("SetLastChannel failed: %v", err)
	}

	// Simulate a crash scenario by manually creating a corrupted temp file
	tempFile := filepath.Join(tmpDir, "state", "state.json.tmp")
	err = os.WriteFile(tempFile, []byte("corrupted data"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Verify that the original state is still intact
	lastChannel := sm.GetLastChannel()
	if lastChannel != "initial-channel" {
		t.Errorf("Expected channel 'initial-channel' after corrupted temp file, got '%s'", lastChannel)
	}

	// Clean up the temp file manually
	os.Remove(tempFile)

	// Now do a proper save
	err = sm.SetLastChannel("new-channel")
	if err != nil {
		t.Fatalf("SetLastChannel failed: %v", err)
	}

	// Verify the new state was saved
	if sm.GetLastChannel() != "new-channel" {
		t.Errorf("Expected channel 'new-channel', got '%s'", sm.GetLastChannel())
	}
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := range 10 {
		go func(idx int) {
			channel := fmt.Sprintf("channel-%d", idx)
			sm.SetLastChannel(channel)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// Verify the final state is consistent
	lastChannel := sm.GetLastChannel()
	if lastChannel == "" {
		t.Error("Expected non-empty channel after concurrent writes")
	}

	// Verify state file is valid JSON
	stateFile := filepath.Join(tmpDir, "state", "state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Errorf("State file contains invalid JSON: %v", err)
	}
}

func TestNewManager_ExistingState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial state
	sm1 := NewManager(tmpDir)
	sm1.SetLastChannel("existing-channel")
	sm1.SetLastChatID("existing-chat-id")

	// Create new manager with same workspace
	sm2 := NewManager(tmpDir)

	// Verify state was loaded
	if sm2.GetLastChannel() != "existing-channel" {
		t.Errorf("Expected channel 'existing-channel', got '%s'", sm2.GetLastChannel())
	}

	if sm2.GetLastChatID() != "existing-chat-id" {
		t.Errorf("Expected chat ID 'existing-chat-id', got '%s'", sm2.GetLastChatID())
	}
}

func TestNewManager_EmptyWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	// Verify default state
	if sm.GetLastChannel() != "" {
		t.Errorf("Expected empty channel, got '%s'", sm.GetLastChannel())
	}

	if sm.GetLastChatID() != "" {
		t.Errorf("Expected empty chat ID, got '%s'", sm.GetLastChatID())
	}

	if !sm.GetTimestamp().IsZero() {
		t.Error("Expected zero timestamp for new state")
	}
}

func TestNewManager_MkdirFailureCrashes(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		tmpDir := os.Getenv("CRASH_DIR")

		statePath := filepath.Join(tmpDir, "state")
		if err := os.WriteFile(statePath, []byte("I'm a file, not a folder"), 0o644); err != nil {
			fmt.Printf("setup failed: %v", err)
			os.Exit(0)
		}

		NewManager(tmpDir)
		os.Exit(0)
	}

	tmpDir, err := os.MkdirTemp("", "state-crash-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command(os.Args[0], "-test.run=TestNewManager_MkdirFailureCrashes")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1", "CRASH_DIR="+tmpDir)

	err = cmd.Run()

	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return
	}

	t.Fatalf("The process ended without error, a crash was expected via os.Exit(1). Err: %v", err)
}

func TestSetGetSessionOverride(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	if got := sm.GetSessionOverride("telegram:123"); got != "" {
		t.Errorf("Expected empty override, got %q", got)
	}

	if err := sm.SetSessionOverride("telegram:123", "agent:main:telegram:direct:123:named:project-a"); err != nil {
		t.Fatalf("SetSessionOverride failed: %v", err)
	}

	got := sm.GetSessionOverride("telegram:123")
	if got != "agent:main:telegram:direct:123:named:project-a" {
		t.Errorf("unexpected override: %q", got)
	}

	if got2 := sm.GetSessionOverride("telegram:456"); got2 != "" {
		t.Errorf("expected empty override for different sender, got %q", got2)
	}
}

func TestClearSessionOverride(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	sm.SetSessionOverride("telegram:123", "agent:main:main")

	if err := sm.ClearSessionOverride("telegram:123"); err != nil {
		t.Fatalf("ClearSessionOverride failed: %v", err)
	}

	if got := sm.GetSessionOverride("telegram:123"); got != "" {
		t.Errorf("expected empty override after clear, got %q", got)
	}

	if err := sm.ClearSessionOverride("nonexistent:sender"); err != nil {
		t.Errorf("ClearSessionOverride of nonexistent sender returned error: %v", err)
	}
}

func TestSessionOverride_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm1 := NewManager(tmpDir)
	sm1.SetSessionOverride("telegram:999", "agent:main:telegram:direct:999:named:special")

	sm2 := NewManager(tmpDir)
	got := sm2.GetSessionOverride("telegram:999")
	if got != "agent:main:telegram:direct:999:named:special" {
		t.Errorf("override not persisted: got %q", got)
	}
}
