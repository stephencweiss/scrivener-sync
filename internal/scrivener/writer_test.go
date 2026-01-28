package scrivener

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyTestProject creates a temporary copy of the test project for modification.
func copyTestProject(t *testing.T) string {
	t.Helper()

	srcDir := filepath.Join(testdataDir, "sample.scriv")
	tmpDir, err := os.MkdirTemp("", "scriv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dstDir := filepath.Join(tmpDir, "sample.scriv")

	// Copy directory recursively
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

	return dstDir
}

func TestWriter_CreateDocument(t *testing.T) {
	projectPath := copyTestProject(t)

	writer, err := NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Find Draft folder
	draftUUID, err := writer.FindFolderByTitle("Draft")
	if err != nil {
		t.Fatalf("Failed to find Draft folder: %v", err)
	}

	// Create a new document
	newUUID, err := writer.CreateDocument("New Chapter", "This is new content.", draftUUID, true)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	if newUUID == "" {
		t.Error("New document should have UUID")
	}

	// Save and reload to verify
	err = writer.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Read back and verify
	reader, err := NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatalf("Failed to read documents: %v", err)
	}

	// Find the new document
	var findDoc func([]*Document, string) *Document
	findDoc = func(docs []*Document, title string) *Document {
		for _, doc := range docs {
			if doc.Title == title {
				return doc
			}
			if found := findDoc(doc.Children, title); found != nil {
				return found
			}
		}
		return nil
	}

	newDoc := findDoc(docs, "New Chapter")
	if newDoc == nil {
		t.Fatal("New document not found after save")
	}

	if newDoc.UUID != newUUID {
		t.Errorf("UUID mismatch: expected %s, got %s", newUUID, newDoc.UUID)
	}
}

func TestWriter_CreateFolder(t *testing.T) {
	projectPath := copyTestProject(t)

	writer, err := NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Find Research folder
	researchUUID, err := writer.FindFolderByTitle("Research")
	if err != nil {
		t.Fatalf("Failed to find Research folder: %v", err)
	}

	// Create a new folder
	newUUID, err := writer.CreateFolder("Notes", researchUUID)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	if newUUID == "" {
		t.Error("New folder should have UUID")
	}

	// Save and verify
	err = writer.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Read back and verify
	reader, err := NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatalf("Failed to read documents: %v", err)
	}

	var findDoc func([]*Document, string) *Document
	findDoc = func(docs []*Document, title string) *Document {
		for _, doc := range docs {
			if doc.Title == title {
				return doc
			}
			if found := findDoc(doc.Children, title); found != nil {
				return found
			}
		}
		return nil
	}

	newFolder := findDoc(docs, "Notes")
	if newFolder == nil {
		t.Fatal("New folder not found after save")
	}

	if !newFolder.IsFolder() {
		t.Error("Notes should be a folder")
	}
}

func TestWriter_UpdateContent(t *testing.T) {
	projectPath := copyTestProject(t)

	writer, err := NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Update content of existing document
	err = writer.UpdateDocumentContent("DOC-UUID-0001", "Updated content here.", true)
	if err != nil {
		t.Fatalf("Failed to update content: %v", err)
	}

	// Read the content file directly
	contentPath := filepath.Join(projectPath, "Files", "Data", "DOC-UUID-0001", "content.rtf")
	data, err := os.ReadFile(contentPath)
	if err != nil {
		t.Fatalf("Failed to read content file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Updated content") {
		t.Errorf("Content not updated: %s", content)
	}
	if !strings.HasPrefix(content, `{\rtf1\ansi`) {
		t.Error("Content should be RTF format")
	}
}

func TestWriter_PreservesProjectAttrs(t *testing.T) {
	projectPath := copyTestProject(t)

	writer, err := NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Make a modification to trigger save
	_, err = writer.CreateDocument("Test Doc", "Test content", "", true)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	err = writer.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Read the saved XML
	scrivxPath := filepath.Join(projectPath, "sample.scrivx")
	data, err := os.ReadFile(scrivxPath)
	if err != nil {
		t.Fatalf("Failed to read scrivx: %v", err)
	}

	content := string(data)

	// Check that key attributes are preserved
	checks := []string{
		`Identifier="TEST-PROJECT-ID"`,
		`Version="2.0"`,
		`Creator="SCRMAC-3.5.2-17487"`,
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Missing attribute: %s", check)
		}
	}
}

func TestWriter_PreservesSections(t *testing.T) {
	projectPath := copyTestProject(t)

	writer, err := NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Make a modification to trigger save
	_, err = writer.CreateDocument("Test Doc", "Test content", "", true)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	err = writer.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Read the saved XML
	scrivxPath := filepath.Join(projectPath, "sample.scrivx")
	data, err := os.ReadFile(scrivxPath)
	if err != nil {
		t.Fatalf("Failed to read scrivx: %v", err)
	}

	content := string(data)

	// Check that sections are preserved
	sections := []string{
		"<Collections>",
		"<LabelSettings>",
		"<StatusSettings>",
		"<ProjectTargets",
		"<PrintSettings",
	}

	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("Missing section: %s", section)
		}
	}
}

func TestWriter_GeneratesUUID(t *testing.T) {
	projectPath := copyTestProject(t)

	writer, err := NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Create multiple documents and check UUIDs are unique
	uuids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		uuid, err := writer.CreateDocument("Test", "Content", "", true)
		if err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		if uuids[uuid] {
			t.Errorf("Duplicate UUID generated: %s", uuid)
		}
		uuids[uuid] = true

		// UUID should be uppercase
		if uuid != strings.ToUpper(uuid) {
			t.Errorf("UUID should be uppercase: %s", uuid)
		}
	}
}

func TestWriter_TimestampFormat(t *testing.T) {
	projectPath := copyTestProject(t)

	writer, err := NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Create a document
	_, err = writer.CreateDocument("Timestamp Test", "Content", "", true)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	err = writer.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Read the saved XML
	scrivxPath := filepath.Join(projectPath, "sample.scrivx")
	data, err := os.ReadFile(scrivxPath)
	if err != nil {
		t.Fatalf("Failed to read scrivx: %v", err)
	}

	// Parse and check timestamp format
	var project XMLProject
	err = xml.Unmarshal(data, &project)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Modified should be in format "2006-01-02 15:04:05 -0700"
	if !strings.Contains(project.Modified, "-") || !strings.Contains(project.Modified, ":") {
		t.Errorf("Invalid timestamp format: %s", project.Modified)
	}

	// Should have timezone offset
	if !strings.Contains(project.Modified, "-0") && !strings.Contains(project.Modified, "+0") {
		t.Errorf("Timestamp should have timezone offset: %s", project.Modified)
	}
}

func TestWriter_FindFolderByTitle(t *testing.T) {
	projectPath := copyTestProject(t)

	writer, err := NewWriter(projectPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Find existing folders
	tests := []struct {
		title    string
		expected bool
	}{
		{"Draft", true},
		{"Research", true},
		{"Characters", true}, // Nested folder
		{"Nonexistent", false},
	}

	for _, tc := range tests {
		uuid, err := writer.FindFolderByTitle(tc.title)
		if tc.expected && err != nil {
			t.Errorf("Expected to find '%s': %v", tc.title, err)
		}
		if !tc.expected && err == nil {
			t.Errorf("Should not find '%s', got UUID: %s", tc.title, uuid)
		}
	}
}
