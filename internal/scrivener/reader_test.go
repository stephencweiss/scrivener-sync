package scrivener

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testdataDir = filepath.Join("..", "..", "testdata")

func TestReadProject_ParsesXML(t *testing.T) {
	projectPath := filepath.Join(testdataDir, "sample.scriv")

	reader, err := NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatalf("Failed to read documents: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document")
	}
}

func TestReadProject_PreservesAttributes(t *testing.T) {
	projectPath := filepath.Join(testdataDir, "sample.scriv")

	reader, err := NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	// Access the internal project XML
	if reader.project == nil {
		t.Fatal("Project should be loaded")
	}

	if reader.project.Identifier == "" {
		t.Error("Project Identifier should be preserved")
	}
	if reader.project.Version == "" {
		t.Error("Project Version should be preserved")
	}
	if reader.project.Creator == "" {
		t.Error("Project Creator should be preserved")
	}
}

func TestReadProject_FolderTypes(t *testing.T) {
	projectPath := filepath.Join(testdataDir, "sample.scriv")

	reader, err := NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatalf("Failed to read documents: %v", err)
	}

	// Find folders by type
	var draftFound, researchFound, trashFound bool
	var checkFolders func([]*Document)
	checkFolders = func(docs []*Document) {
		for _, doc := range docs {
			if doc.Title == "Draft" && doc.IsFolder() {
				draftFound = true
			}
			if doc.Title == "Research" && doc.IsFolder() {
				researchFound = true
			}
			if doc.Title == "Trash" && doc.IsFolder() {
				trashFound = true
			}
			checkFolders(doc.Children)
		}
	}
	checkFolders(docs)

	if !draftFound {
		t.Error("DraftFolder should be recognized as a folder")
	}
	if !researchFound {
		t.Error("ResearchFolder should be recognized as a folder")
	}
	if !trashFound {
		t.Error("TrashFolder should be recognized as a folder")
	}
}

func TestReadProject_NestedChildren(t *testing.T) {
	projectPath := filepath.Join(testdataDir, "sample.scriv")

	reader, err := NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatalf("Failed to read documents: %v", err)
	}

	// Find the Research folder and check for nested Characters folder
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

	research := findDoc(docs, "Research")
	if research == nil {
		t.Fatal("Research folder not found")
	}

	characters := findDoc(research.Children, "Characters")
	if characters == nil {
		t.Fatal("Characters folder not found under Research")
	}

	hero := findDoc(characters.Children, "Hero")
	if hero == nil {
		t.Fatal("Hero document not found under Characters")
	}
}

func TestReadProject_ReadsContent(t *testing.T) {
	projectPath := filepath.Join(testdataDir, "sample.scriv")

	reader, err := NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatalf("Failed to read documents: %v", err)
	}

	// Find Chapter One and check content
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

	chapterOne := findDoc(docs, "Chapter One")
	if chapterOne == nil {
		t.Fatal("Chapter One not found")
	}

	if chapterOne.Content == "" {
		t.Error("Chapter One should have content")
	}

	// Content should be converted from RTF
	if strings.Contains(chapterOne.Content, "\\rtf") {
		t.Error("Content should not contain raw RTF")
	}
	if !strings.Contains(chapterOne.Content, "story begins") {
		t.Errorf("Content should contain 'story begins', got: %s", chapterOne.Content)
	}
}

func TestReadProject_DocumentUUID(t *testing.T) {
	projectPath := filepath.Join(testdataDir, "sample.scriv")

	reader, err := NewReader(projectPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	docs, err := reader.GetBinderStructure()
	if err != nil {
		t.Fatalf("Failed to read documents: %v", err)
	}

	// Find a document and check UUID
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

	chapterOne := findDoc(docs, "Chapter One")
	if chapterOne == nil {
		t.Fatal("Chapter One not found")
	}

	if chapterOne.UUID == "" {
		t.Error("Document should have UUID")
	}
	if chapterOne.UUID != "DOC-UUID-0001" {
		t.Errorf("Expected UUID 'DOC-UUID-0001', got '%s'", chapterOne.UUID)
	}
}

func TestReadProject_NotFound(t *testing.T) {
	_, err := NewReader("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
}

func TestReadProject_InvalidProject(t *testing.T) {
	// Create a temp directory without .scrivx file
	tmpDir, err := os.MkdirTemp("", "invalid-scriv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = NewReader(tmpDir)
	if err == nil {
		t.Error("Expected error for project without .scrivx file")
	}
}
