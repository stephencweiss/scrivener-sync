package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestState_NewState(t *testing.T) {
	state := NewState("/tmp/test-state.json")

	if state == nil {
		t.Fatal("NewState should return non-nil")
	}
	if state.Files == nil {
		t.Error("Files map should be initialized")
	}
	if state.DeletedFiles == nil {
		t.Error("DeletedFiles map should be initialized")
	}
	if len(state.Files) != 0 {
		t.Error("Files map should be empty")
	}
}

func TestState_SaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "test-state.json")

	// Create and populate state
	state := NewState(statePath)
	state.ScrivPath = "/path/to/project.scriv"
	state.RecordFile("/path/to/file.md", "UUID-123", "hash123", time.Now())

	// Save
	err = state.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("State file not created: %v", err)
	}

	// Load
	loaded, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if loaded.ScrivPath != "/path/to/project.scriv" {
		t.Errorf("ScrivPath not preserved: %s", loaded.ScrivPath)
	}

	if len(loaded.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(loaded.Files))
	}

	fs := loaded.GetFileState("/path/to/file.md")
	if fs == nil {
		t.Fatal("File state not found")
	}
	if fs.ScrivUUID != "UUID-123" {
		t.Errorf("UUID not preserved: %s", fs.ScrivUUID)
	}
}

func TestState_LoadNonexistent(t *testing.T) {
	state, err := LoadState("/nonexistent/path/state.json")
	if err != nil {
		t.Fatalf("Should not error for nonexistent file: %v", err)
	}
	if state == nil {
		t.Fatal("Should return new state for nonexistent file")
	}
	if len(state.Files) != 0 {
		t.Error("New state should have empty Files")
	}
}

func TestState_RecordFile(t *testing.T) {
	state := NewState("/tmp/test.json")

	now := time.Now()
	state.RecordFile("/test/file.md", "UUID-ABC", "contenthash", now)

	fs := state.GetFileState("/test/file.md")
	if fs == nil {
		t.Fatal("File not recorded")
	}
	if fs.ScrivUUID != "UUID-ABC" {
		t.Errorf("Wrong UUID: %s", fs.ScrivUUID)
	}
	if fs.ContentHash != "contenthash" {
		t.Errorf("Wrong hash: %s", fs.ContentHash)
	}
}

func TestState_RemoveFile(t *testing.T) {
	state := NewState("/tmp/test.json")

	// Record a file
	state.RecordFile("/test/file.md", "UUID-ABC", "hash", time.Now())

	// Remove it
	state.RemoveFile("/test/file.md")

	// Should be gone from Files
	if state.GetFileState("/test/file.md") != nil {
		t.Error("File should be removed from Files")
	}

	// Should be in DeletedFiles
	if state.GetDeletedFileState("/test/file.md") == nil {
		t.Error("File should be in DeletedFiles")
	}

	// WasPreviouslySynced should still return true
	if !state.WasPreviouslySynced("/test/file.md") {
		t.Error("WasPreviouslySynced should return true for deleted file")
	}
}

func TestState_DetectConflict_NewFile(t *testing.T) {
	state := NewState("/tmp/test.json")

	conflict := state.DetectConflict("/new/file.md", "hash1", "UUID", "hash2")
	if conflict != ConflictNewFile {
		t.Errorf("Expected ConflictNewFile, got %s", conflict)
	}
}

func TestState_DetectConflict_None(t *testing.T) {
	state := NewState("/tmp/test.json")
	state.RecordFile("/test/file.md", "UUID-ABC", "samehash", time.Now())

	conflict := state.DetectConflict("/test/file.md", "samehash", "UUID-ABC", "samehash")
	if conflict != ConflictNone {
		t.Errorf("Expected ConflictNone, got %s", conflict)
	}
}

func TestState_DetectConflict_MarkdownOnly(t *testing.T) {
	state := NewState("/tmp/test.json")
	state.RecordFile("/test/file.md", "UUID-ABC", "oldhash", time.Now())

	// Markdown changed, Scrivener unchanged
	conflict := state.DetectConflict("/test/file.md", "newhash", "UUID-ABC", "oldhash")
	if conflict != ConflictMarkdownOnly {
		t.Errorf("Expected ConflictMarkdownOnly, got %s", conflict)
	}
}

func TestState_DetectConflict_ScrivenerOnly(t *testing.T) {
	state := NewState("/tmp/test.json")
	state.RecordFile("/test/file.md", "UUID-ABC", "oldhash", time.Now())

	// Scrivener changed, markdown unchanged
	conflict := state.DetectConflict("/test/file.md", "oldhash", "UUID-ABC", "newhash")
	if conflict != ConflictScrivenerOnly {
		t.Errorf("Expected ConflictScrivenerOnly, got %s", conflict)
	}
}

func TestState_DetectConflict_Both(t *testing.T) {
	state := NewState("/tmp/test.json")
	state.RecordFile("/test/file.md", "UUID-ABC", "oldhash", time.Now())

	// Both changed
	conflict := state.DetectConflict("/test/file.md", "newhash1", "UUID-ABC", "newhash2")
	if conflict != ConflictBoth {
		t.Errorf("Expected ConflictBoth, got %s", conflict)
	}
}

func TestState_GetUUIDForPath(t *testing.T) {
	state := NewState("/tmp/test.json")
	state.RecordFile("/test/file.md", "UUID-123", "hash", time.Now())

	uuid := state.GetUUIDForPath("/test/file.md")
	if uuid != "UUID-123" {
		t.Errorf("Expected UUID-123, got %s", uuid)
	}

	uuid = state.GetUUIDForPath("/nonexistent.md")
	if uuid != "" {
		t.Errorf("Expected empty string for nonexistent path, got %s", uuid)
	}
}

func TestState_GetPathForUUID(t *testing.T) {
	state := NewState("/tmp/test.json")
	state.RecordFile("/test/file.md", "UUID-123", "hash", time.Now())

	path := state.GetPathForUUID("UUID-123")
	if path != "/test/file.md" {
		t.Errorf("Expected /test/file.md, got %s", path)
	}

	path = state.GetPathForUUID("NONEXISTENT")
	if path != "" {
		t.Errorf("Expected empty string for nonexistent UUID, got %s", path)
	}
}

func TestState_AllTrackedPaths(t *testing.T) {
	state := NewState("/tmp/test.json")
	state.RecordFile("/test/a.md", "UUID-A", "hash", time.Now())
	state.RecordFile("/test/b.md", "UUID-B", "hash", time.Now())
	state.RecordFile("/test/c.md", "UUID-C", "hash", time.Now())

	paths := state.AllTrackedPaths()
	if len(paths) != 3 {
		t.Errorf("Expected 3 paths, got %d", len(paths))
	}

	// Check all paths are present
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}
	expected := []string{"/test/a.md", "/test/b.md", "/test/c.md"}
	for _, e := range expected {
		if !pathSet[e] {
			t.Errorf("Missing path: %s", e)
		}
	}
}

func TestState_AllTrackedUUIDs(t *testing.T) {
	state := NewState("/tmp/test.json")
	state.RecordFile("/test/a.md", "UUID-A", "hash", time.Now())
	state.RecordFile("/test/b.md", "UUID-B", "hash", time.Now())

	uuids := state.AllTrackedUUIDs()
	if len(uuids) != 2 {
		t.Errorf("Expected 2 UUIDs, got %d", len(uuids))
	}
}

func TestState_UpdateLastSync(t *testing.T) {
	state := NewState("/tmp/test.json")

	if state.LastSync != nil {
		t.Error("LastSync should be nil initially")
	}

	state.UpdateLastSync()

	if state.LastSync == nil {
		t.Error("LastSync should be set after UpdateLastSync")
	}

	// Should be recent
	if time.Since(*state.LastSync) > time.Second {
		t.Error("LastSync should be recent")
	}
}

func TestState_RecordFileRemovesFromDeleted(t *testing.T) {
	state := NewState("/tmp/test.json")

	// Record, remove, then re-record
	state.RecordFile("/test/file.md", "UUID-1", "hash1", time.Now())
	state.RemoveFile("/test/file.md")

	// Should be in DeletedFiles
	if state.GetDeletedFileState("/test/file.md") == nil {
		t.Error("File should be in DeletedFiles")
	}

	// Re-record with new info
	state.RecordFile("/test/file.md", "UUID-2", "hash2", time.Now())

	// Should be back in Files
	if state.GetFileState("/test/file.md") == nil {
		t.Error("File should be in Files")
	}

	// Should be removed from DeletedFiles
	if state.GetDeletedFileState("/test/file.md") != nil {
		t.Error("File should be removed from DeletedFiles")
	}
}
