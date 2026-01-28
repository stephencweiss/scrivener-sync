package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sweiss/harcroft/internal/config"
	"github.com/sweiss/harcroft/internal/scrivener"
)

var testdataDir = filepath.Join("..", "..", "testdata")

// copyTestProject creates a temporary copy of the sample Scrivener project.
func copyTestProject(t *testing.T) string {
	t.Helper()

	srcDir := filepath.Join(testdataDir, "sample.scriv")
	tmpDir, err := os.MkdirTemp("", "sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "sample.scriv")

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(srcDir, path)
		dstPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})

	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to copy test project: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

// TestIntegration_ReadWriteRoundtrip tests reading and writing preserves content.
func TestIntegration_ReadWriteRoundtrip(t *testing.T) {
	tmpDir := copyTestProject(t)
	projectPath := filepath.Join(tmpDir, "sample.scriv")

	// Read original documents
	reader, err := scrivener.NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatalf("Failed to read docs: %v", err)
	}

	// Count original docs
	originalCount := countDocs(docs)

	// Create writer and add a document
	writer, err := scrivener.NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	draftUUID, _ := writer.FindFolderByTitle("Draft")
	_, err = writer.CreateDocument("Test Chapter", "Test content", draftUUID, true)
	if err != nil {
		t.Fatalf("Failed to create doc: %v", err)
	}

	err = writer.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Read again and verify
	reader2, _ := scrivener.NewReader(projectPath)
	docs2, _ := reader2.GetBinderStructure()

	newCount := countDocs(docs2)
	if newCount != originalCount+1 {
		t.Errorf("Expected %d docs, got %d", originalCount+1, newCount)
	}
}

func countDocs(docs []*scrivener.Document) int {
	count := 0
	for _, doc := range docs {
		count++
		count += countDocs(doc.Children)
	}
	return count
}

// TestIntegration_StateTracksChanges tests that state correctly tracks document changes.
func TestIntegration_StateTracksChanges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-int-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "state.json")
	state := NewState(statePath)

	// Simulate initial sync
	state.RecordFile("/docs/chapter1.md", "UUID-1", "hash-initial", time.Now())
	state.Save()

	// Simulate content change
	conflict := state.DetectConflict("/docs/chapter1.md", "hash-new", "UUID-1", "hash-initial")
	if conflict != ConflictMarkdownOnly {
		t.Errorf("Expected markdown-only change, got %s", conflict)
	}

	// Simulate Scrivener change
	conflict = state.DetectConflict("/docs/chapter1.md", "hash-initial", "UUID-1", "hash-scriv-new")
	if conflict != ConflictScrivenerOnly {
		t.Errorf("Expected scrivener-only change, got %s", conflict)
	}

	// Simulate both changed
	conflict = state.DetectConflict("/docs/chapter1.md", "hash-md-new", "UUID-1", "hash-scriv-new")
	if conflict != ConflictBoth {
		t.Errorf("Expected both changed, got %s", conflict)
	}
}

// TestIntegration_XMLPreservation tests that Scrivener XML structure is preserved.
func TestIntegration_XMLPreservation(t *testing.T) {
	tmpDir := copyTestProject(t)
	projectPath := filepath.Join(tmpDir, "sample.scriv")

	// Read original XML
	scrivxPath := filepath.Join(projectPath, "sample.scrivx")
	originalData, err := os.ReadFile(scrivxPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create writer and make a change
	writer, err := scrivener.NewWriter(projectPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = writer.CreateDocument("Preservation Test", "Content", "", true)
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Save()
	if err != nil {
		t.Fatal(err)
	}

	// Read saved XML
	savedData, err := os.ReadFile(scrivxPath)
	if err != nil {
		t.Fatal(err)
	}

	savedStr := string(savedData)

	// Verify key elements are preserved
	elementsToCheck := []string{
		`Identifier="TEST-PROJECT-ID"`,
		`Version="2.0"`,
		"<Collections>",
		"<LabelSettings>",
		"<StatusSettings>",
		"<ProjectTargets",
		"<PrintSettings",
	}

	for _, elem := range elementsToCheck {
		if !strings.Contains(savedStr, elem) {
			t.Errorf("Missing element after save: %s", elem)
			t.Logf("Original had it: %v", strings.Contains(string(originalData), elem))
		}
	}
}

// TestIntegration_ContentConversion tests RTF/markdown conversion in context.
func TestIntegration_ContentConversion(t *testing.T) {
	tmpDir := copyTestProject(t)
	projectPath := filepath.Join(tmpDir, "sample.scriv")

	// Read a document's content
	reader, err := scrivener.NewReader(projectPath)
	if err != nil {
		t.Fatal(err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatal(err)
	}

	// Find Chapter One
	var chapter *scrivener.Document
	var findChapter func([]*scrivener.Document)
	findChapter = func(docs []*scrivener.Document) {
		for _, doc := range docs {
			if doc.Title == "Chapter One" {
				chapter = doc
				return
			}
			findChapter(doc.Children)
		}
	}
	findChapter(docs)

	if chapter == nil {
		t.Fatal("Chapter One not found")
	}

	// Content should be converted from RTF (no RTF artifacts)
	if strings.Contains(chapter.Content, "\\rtf") {
		t.Error("Content should not contain raw RTF")
	}
	if strings.Contains(chapter.Content, "\\pard") {
		t.Error("Content should not contain \\pard")
	}
	if !strings.Contains(chapter.Content, "story begins") {
		t.Error("Content should contain actual text")
	}
}

// TestIntegration_FolderMappingResolution tests that folder mappings work correctly.
func TestIntegration_FolderMappingResolution(t *testing.T) {
	tmpDir := copyTestProject(t)
	projectPath := filepath.Join(tmpDir, "sample.scriv")

	writer, err := scrivener.NewWriter(projectPath)
	if err != nil {
		t.Fatal(err)
	}

	// Test finding different folder types
	folderTests := []struct {
		name     string
		expected bool
	}{
		{"Draft", true},
		{"Research", true},
		{"Characters", true},
		{"Nonexistent", false},
	}

	for _, tc := range folderTests {
		uuid, err := writer.FindFolderByTitle(tc.name)
		found := err == nil && uuid != ""

		if found != tc.expected {
			if tc.expected {
				t.Errorf("Expected to find '%s' but didn't: %v", tc.name, err)
			} else {
				t.Errorf("Expected not to find '%s' but got UUID: %s", tc.name, uuid)
			}
		}
	}
}

// TestIntegration_ProjectConfigCreation tests config creation works.
func TestIntegration_ProjectConfigCreation(t *testing.T) {
	tmpDir := copyTestProject(t)
	projectPath := filepath.Join(tmpDir, "sample.scriv")
	mdPath := filepath.Join(tmpDir, "markdown")
	os.MkdirAll(mdPath, 0755)

	cfg := &config.ProjectConfig{
		ScrivPath: projectPath,
		LocalPath: mdPath,
		FolderMappings: []config.FolderMapping{
			{ScrivenerFolder: "Draft", MarkdownDir: "draft", SyncEnabled: true},
			{ScrivenerFolder: "Research/Characters", MarkdownDir: "characters", SyncEnabled: true},
		},
	}

	// Verify paths resolve correctly
	scrivPath, err := cfg.ScrivenerPath()
	if err != nil {
		t.Errorf("Failed to get Scrivener path: %v", err)
	}
	if scrivPath != projectPath {
		t.Errorf("Scrivener path mismatch: %s vs %s", scrivPath, projectPath)
	}

	mdRoot := cfg.MarkdownPath()
	if mdRoot != mdPath {
		t.Errorf("Markdown path mismatch: %s vs %s", mdRoot, mdPath)
	}
}
