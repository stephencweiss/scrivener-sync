package scrivener

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sweiss/harcroft/internal/rtf"
)

// Writer writes content to Scrivener project files.
type Writer struct {
	scrivPath     string
	projectXML    string
	filesDir      string
	project       *XMLProject
	existingUUIDs map[string]bool
	modified      bool
}

// NewWriter creates a new Writer for the given Scrivener project path.
func NewWriter(scrivPath string) (*Writer, error) {
	// Validate .scriv exists
	info, err := os.Stat(scrivPath)
	if err != nil {
		return nil, fmt.Errorf("scrivener project not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scrivener project must be a directory: %s", scrivPath)
	}

	// Find project.scrivx file
	projectXML := ""
	entries, err := os.ReadDir(scrivPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project directory: %w", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".scrivx") {
			projectXML = filepath.Join(scrivPath, entry.Name())
			break
		}
	}
	if projectXML == "" {
		return nil, fmt.Errorf("no .scrivx file found in %s", scrivPath)
	}

	// Set up filesDir path
	filesDir := filepath.Join(scrivPath, "Files", "Data")

	// Ensure Files/Data directory exists
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	w := &Writer{
		scrivPath:     scrivPath,
		projectXML:    projectXML,
		filesDir:      filesDir,
		existingUUIDs: make(map[string]bool),
	}

	// Load the project XML
	if err := w.loadProject(); err != nil {
		return nil, err
	}

	// Collect existing UUIDs
	w.collectUUIDs(w.project.Binder.Items)

	return w, nil
}

// loadProject parses the project.scrivx XML file.
func (w *Writer) loadProject() error {
	data, err := os.ReadFile(w.projectXML)
	if err != nil {
		return fmt.Errorf("failed to read project file: %w", err)
	}

	w.project = &XMLProject{}
	if err := xml.Unmarshal(data, w.project); err != nil {
		return fmt.Errorf("failed to parse project XML: %w", err)
	}

	return nil
}

// collectUUIDs recursively collects all UUIDs from binder items.
func (w *Writer) collectUUIDs(items []XMLBinderItem) {
	for _, item := range items {
		if item.UUID != "" {
			w.existingUUIDs[item.UUID] = true
		}
		w.collectUUIDs(item.Children)
	}
}

// UpdateDocumentContent updates the content of an existing document.
// When useRTF is true, converts markdown to RTF format for Scrivener.
func (w *Writer) UpdateDocumentContent(docUUID, content string, useRTF bool) error {
	// Determine content path - try new format first
	contentDir := filepath.Join(w.filesDir, docUUID)
	if info, err := os.Stat(contentDir); err == nil && info.IsDir() {
		// New format: Files/Data/{UUID}/content.rtf
		var contentPath string
		var data string
		if useRTF {
			contentPath = filepath.Join(contentDir, "content.rtf")
			data = rtf.MarkdownToRTF(content)
		} else {
			contentPath = filepath.Join(contentDir, "content.txt")
			data = content
		}
		return os.WriteFile(contentPath, []byte(data), 0644)
	}

	// Old format: Files/Data/{UUID}.rtf
	var contentPath string
	var data string
	if useRTF {
		contentPath = filepath.Join(w.filesDir, docUUID+".rtf")
		data = rtf.MarkdownToRTF(content)
	} else {
		contentPath = filepath.Join(w.filesDir, docUUID+".txt")
		data = content
	}
	return os.WriteFile(contentPath, []byte(data), 0644)
}

// CreateFolder creates a new folder in the binder.
func (w *Writer) CreateFolder(title, parentUUID string) (string, error) {
	newUUID := w.generateUUID()
	now := time.Now().Format("2006-01-02 15:04:05 -0700")

	item := XMLBinderItem{
		UUID:         newUUID,
		Type:         "Folder",
		Created:      now,
		Modified:     now,
		Title:        title,
		MetaData:     &XMLMetaData{IncludeInCompile: "Yes"},
		TextSettings: &XMLTextSettings{TextSelection: "0,0"},
	}

	if parentUUID == "" {
		// Add to root binder
		w.project.Binder.Items = append(w.project.Binder.Items, item)
	} else {
		// Add to parent's children
		if !w.addToParent(&w.project.Binder.Items, parentUUID, item) {
			return "", fmt.Errorf("parent UUID not found: %s", parentUUID)
		}
	}

	w.existingUUIDs[newUUID] = true
	w.modified = true
	return newUUID, nil
}

// CreateDocument creates a new document in the binder.
func (w *Writer) CreateDocument(title, content, parentUUID string, useRTF bool) (string, error) {
	newUUID := w.generateUUID()
	now := time.Now().Format("2006-01-02 15:04:05 -0700")

	item := XMLBinderItem{
		UUID:         newUUID,
		Type:         "Text",
		Created:      now,
		Modified:     now,
		Title:        title,
		MetaData:     &XMLMetaData{IncludeInCompile: "Yes"},
		TextSettings: &XMLTextSettings{TextSelection: "0,0"},
	}

	if parentUUID == "" {
		// Add to root binder
		w.project.Binder.Items = append(w.project.Binder.Items, item)
	} else {
		// Add to parent's children
		if !w.addToParent(&w.project.Binder.Items, parentUUID, item) {
			return "", fmt.Errorf("parent UUID not found: %s", parentUUID)
		}
	}

	// Create content directory and file
	contentDir := filepath.Join(w.filesDir, newUUID)
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create content directory: %w", err)
	}

	if err := w.UpdateDocumentContent(newUUID, content, useRTF); err != nil {
		return "", err
	}

	w.existingUUIDs[newUUID] = true
	w.modified = true
	return newUUID, nil
}

// addToParent recursively finds the parent and adds the item to its children.
func (w *Writer) addToParent(items *[]XMLBinderItem, parentUUID string, item XMLBinderItem) bool {
	for i := range *items {
		if (*items)[i].UUID == parentUUID {
			(*items)[i].Children = append((*items)[i].Children, item)
			return true
		}
		if w.addToParent(&(*items)[i].Children, parentUUID, item) {
			return true
		}
	}
	return false
}

// FindFolderByTitle finds a folder by title and returns its UUID.
func (w *Writer) FindFolderByTitle(title string) (string, error) {
	uuid := w.findFolderUUID(w.project.Binder.Items, title)
	if uuid == "" {
		return "", fmt.Errorf("folder not found: %s", title)
	}
	return uuid, nil
}

func (w *Writer) findFolderUUID(items []XMLBinderItem, title string) string {
	lowerTitle := strings.ToLower(title)
	for _, item := range items {
		isFolder := item.Type == "Folder" || item.Type == "DraftFolder" || item.Type == "ResearchFolder"
		if isFolder && strings.ToLower(item.Title) == lowerTitle {
			return item.UUID
		}
		if uuid := w.findFolderUUID(item.Children, title); uuid != "" {
			return uuid
		}
	}
	return ""
}

// Save writes changes back to the project.scrivx file.
func (w *Writer) Save() error {
	if !w.modified {
		return nil
	}

	// Update project modification timestamp and ID
	w.project.Modified = time.Now().Format("2006-01-02 15:04:05 -0700")
	w.project.ModID = strings.ToUpper(uuid.New().String())

	data, err := xml.MarshalIndent(w.project, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal project XML: %w", err)
	}

	// Add XML declaration
	xmlData := []byte(xml.Header + string(data))

	if err := os.WriteFile(w.projectXML, xmlData, 0644); err != nil {
		return fmt.Errorf("failed to write project file: %w", err)
	}

	w.modified = false
	return nil
}

// generateUUID generates a unique UUID that doesn't conflict with existing ones.
func (w *Writer) generateUUID() string {
	for {
		newUUID := uuid.New().String()
		// Scrivener uses uppercase UUIDs
		newUUID = strings.ToUpper(newUUID)
		if !w.existingUUIDs[newUUID] {
			return newUUID
		}
	}
}

// findBinderItem finds a binder item by UUID.
func (w *Writer) findBinderItem(uuid string) *XMLBinderItem {
	return w.findInItems(w.project.Binder.Items, uuid)
}

func (w *Writer) findInItems(items []XMLBinderItem, uuid string) *XMLBinderItem {
	for i := range items {
		if items[i].UUID == uuid {
			return &items[i]
		}
		if found := w.findInItems(items[i].Children, uuid); found != nil {
			return found
		}
	}
	return nil
}
