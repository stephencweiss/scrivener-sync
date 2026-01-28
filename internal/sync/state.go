// Package sync provides bi-directional sync between Scrivener and markdown.
package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sweiss/harcroft/internal/config"
)

// State tracks the sync state between markdown files and Scrivener documents.
type State struct {
	LastSync      *time.Time           `json:"last_sync"`
	Files         map[string]FileState `json:"files"`
	ScrivPath     string               `json:"scriv_path"`
	DeletedFiles  map[string]FileState `json:"deleted_files,omitempty"`
	ConfigVersion string               `json:"config_version"`

	filePath string
}

// FileState represents the sync state of a single file.
type FileState struct {
	ScrivUUID    string `json:"scriv_uuid"`
	ContentHash  string `json:"content_hash"`
	ModifiedTime string `json:"modified_time"`
	LastSynced   string `json:"last_synced"`
}

// ConflictType represents the type of conflict detected during sync.
type ConflictType string

const (
	// ConflictNone indicates no conflict.
	ConflictNone ConflictType = "none"
	// ConflictMarkdownOnly indicates only the markdown file was modified.
	ConflictMarkdownOnly ConflictType = "markdown_modified"
	// ConflictScrivenerOnly indicates only the Scrivener document was modified.
	ConflictScrivenerOnly ConflictType = "scrivener_modified"
	// ConflictBoth indicates both sides were modified.
	ConflictBoth ConflictType = "both_modified"
	// ConflictNewFile indicates a new file that hasn't been synced before.
	ConflictNewFile ConflictType = "new_file"
)

// LoadState reads the state file from the given path.
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewState(path), nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	state := &State{}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	state.filePath = path

	// Initialize maps if nil
	if state.Files == nil {
		state.Files = make(map[string]FileState)
	}
	if state.DeletedFiles == nil {
		state.DeletedFiles = make(map[string]FileState)
	}

	return state, nil
}

// NewState creates a new empty state.
func NewState(path string) *State {
	return &State{
		Files:        make(map[string]FileState),
		DeletedFiles: make(map[string]FileState),
		filePath:     path,
	}
}

// LoadStateForAlias loads the state file for a project alias from ~/.scriv-sync/state/<alias>.json.
func LoadStateForAlias(alias string) (*State, error) {
	statePath, err := config.StatePath(alias)
	if err != nil {
		return nil, err
	}

	// Ensure state directory exists
	stateDir := filepath.Dir(statePath)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	return LoadState(statePath)
}

// Save writes the state to its file.
func (s *State) Save() error {
	if s.filePath == "" {
		return fmt.Errorf("state file path not set")
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// RecordFile records the sync state for a file.
func (s *State) RecordFile(mdPath, scrivUUID, hash string, modified time.Time) {
	now := time.Now().Format(time.RFC3339)
	s.Files[mdPath] = FileState{
		ScrivUUID:    scrivUUID,
		ContentHash:  hash,
		ModifiedTime: modified.Format(time.RFC3339),
		LastSynced:   now,
	}

	// Remove from deleted files if it was there
	delete(s.DeletedFiles, mdPath)
}

// RemoveFile removes a file from the state and records it as deleted.
func (s *State) RemoveFile(mdPath string) {
	if fs, exists := s.Files[mdPath]; exists {
		s.DeletedFiles[mdPath] = fs
		delete(s.Files, mdPath)
	}
}

// GetFileState returns the state for a file, or nil if not tracked.
func (s *State) GetFileState(mdPath string) *FileState {
	if fs, exists := s.Files[mdPath]; exists {
		return &fs
	}
	return nil
}

// WasPreviouslySynced returns true if the file was synced before (and possibly deleted).
func (s *State) WasPreviouslySynced(mdPath string) bool {
	_, inFiles := s.Files[mdPath]
	_, inDeleted := s.DeletedFiles[mdPath]
	return inFiles || inDeleted
}

// GetDeletedFileState returns the state for a deleted file, or nil if not found.
func (s *State) GetDeletedFileState(mdPath string) *FileState {
	if fs, exists := s.DeletedFiles[mdPath]; exists {
		return &fs
	}
	return nil
}

// DetectConflict determines the conflict type between markdown and Scrivener versions.
func (s *State) DetectConflict(mdPath, mdHash, scrivUUID, scrivHash string) ConflictType {
	fs := s.GetFileState(mdPath)
	if fs == nil {
		// Check if it was deleted
		if dfs := s.GetDeletedFileState(mdPath); dfs != nil {
			// File was previously synced but deleted from one side
			return ConflictBoth // Treat as conflict - needs user decision
		}
		return ConflictNewFile
	}

	mdChanged := fs.ContentHash != mdHash
	scrivChanged := fs.ContentHash != scrivHash

	if mdChanged && scrivChanged {
		return ConflictBoth
	}
	if mdChanged {
		return ConflictMarkdownOnly
	}
	if scrivChanged {
		return ConflictScrivenerOnly
	}

	return ConflictNone
}

// SetScrivPath sets the Scrivener project path.
func (s *State) SetScrivPath(path string) {
	s.ScrivPath = path
}

// UpdateLastSync updates the last sync timestamp to now.
func (s *State) UpdateLastSync() {
	now := time.Now()
	s.LastSync = &now
}

// GetUUIDForPath returns the Scrivener UUID for a markdown path, or empty string if not found.
func (s *State) GetUUIDForPath(mdPath string) string {
	if fs := s.GetFileState(mdPath); fs != nil {
		return fs.ScrivUUID
	}
	return ""
}

// GetPathForUUID returns the markdown path for a Scrivener UUID, or empty string if not found.
func (s *State) GetPathForUUID(uuid string) string {
	for path, fs := range s.Files {
		if fs.ScrivUUID == uuid {
			return path
		}
	}
	return ""
}

// AllTrackedPaths returns all currently tracked markdown paths.
func (s *State) AllTrackedPaths() []string {
	paths := make([]string, 0, len(s.Files))
	for path := range s.Files {
		paths = append(paths, path)
	}
	return paths
}

// AllTrackedUUIDs returns all currently tracked Scrivener UUIDs.
func (s *State) AllTrackedUUIDs() []string {
	uuids := make([]string, 0, len(s.Files))
	for _, fs := range s.Files {
		uuids = append(uuids, fs.ScrivUUID)
	}
	return uuids
}
