package scrivener

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sweiss/harcroft/internal/rtf"
)

// Reader reads and parses Scrivener project files.
type Reader struct {
	scrivPath  string
	projectXML string
	filesDir   string
	project    *XMLProject
}

// NewReader creates a new Reader for the given Scrivener project path.
func NewReader(scrivPath string) (*Reader, error) {
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

	// Set up filesDir path (Files/Data)
	filesDir := filepath.Join(scrivPath, "Files", "Data")

	r := &Reader{
		scrivPath:  scrivPath,
		projectXML: projectXML,
		filesDir:   filesDir,
	}

	// Parse the project XML
	if err := r.loadProject(); err != nil {
		return nil, err
	}

	return r, nil
}

// loadProject parses the project.scrivx XML file.
func (r *Reader) loadProject() error {
	data, err := os.ReadFile(r.projectXML)
	if err != nil {
		return fmt.Errorf("failed to read project file: %w", err)
	}

	r.project = &XMLProject{}
	if err := xml.Unmarshal(data, r.project); err != nil {
		return fmt.Errorf("failed to parse project XML: %w", err)
	}

	return nil
}

// GetBinderStructure returns the complete document tree from the binder.
func (r *Reader) GetBinderStructure() ([]*Document, error) {
	var docs []*Document
	for _, item := range r.project.Binder.Items {
		doc, err := r.parseBinderItem(item)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

// GetTopLevelFolders returns only the top-level folders from the binder.
func (r *Reader) GetTopLevelFolders() ([]*Document, error) {
	docs, err := r.GetBinderStructure()
	if err != nil {
		return nil, err
	}

	var folders []*Document
	for _, doc := range docs {
		if doc.IsFolder() {
			folders = append(folders, doc)
		}
	}
	return folders, nil
}

// FindFolderByTitle finds a folder by its title (case-insensitive).
func (r *Reader) FindFolderByTitle(title string) (*Document, error) {
	docs, err := r.GetBinderStructure()
	if err != nil {
		return nil, err
	}
	return r.findFolderInDocs(docs, title), nil
}

func (r *Reader) findFolderInDocs(docs []*Document, title string) *Document {
	lowerTitle := strings.ToLower(title)
	for _, doc := range docs {
		if doc.IsFolder() && strings.ToLower(doc.Title) == lowerTitle {
			return doc
		}
		if found := r.findFolderInDocs(doc.Children, title); found != nil {
			return found
		}
	}
	return nil
}

// GetAllDocuments returns a flattened list of all documents (not folders).
func (r *Reader) GetAllDocuments() ([]*Document, error) {
	docs, err := r.GetBinderStructure()
	if err != nil {
		return nil, err
	}
	return r.flattenDocs(docs, false), nil
}

func (r *Reader) flattenDocs(docs []*Document, includeFolders bool) []*Document {
	var result []*Document
	for _, doc := range docs {
		if includeFolders || !doc.IsFolder() {
			result = append(result, doc)
		}
		result = append(result, r.flattenDocs(doc.Children, includeFolders)...)
	}
	return result
}

// parseBinderItem converts an XMLBinderItem to a Document.
func (r *Reader) parseBinderItem(item XMLBinderItem) (*Document, error) {
	if item.UUID == "" {
		return nil, nil
	}

	docType := "document"
	if item.Type == "Folder" || item.Type == "DraftFolder" || item.Type == "ResearchFolder" || item.Type == "TrashFolder" {
		docType = "folder"
	}

	content, err := r.readDocumentContent(item.UUID)
	if err != nil {
		// Not all items have content (e.g., folders)
		content = ""
	}

	doc := &Document{
		UUID:     item.UUID,
		Title:    item.Title,
		Content:  content,
		DocType:  docType,
		Modified: r.getModificationTime(item.UUID),
	}

	// Parse children recursively
	for _, child := range item.Children {
		childDoc, err := r.parseBinderItem(child)
		if err != nil {
			return nil, err
		}
		if childDoc != nil {
			doc.Children = append(doc.Children, childDoc)
		}
	}

	return doc, nil
}

// readDocumentContent reads the content of a document by its UUID.
func (r *Reader) readDocumentContent(uuid string) (string, error) {
	// Scrivener 3 stores documents in Files/Data/{UUID}/content.rtf
	// Try the new format first
	contentPath := filepath.Join(r.filesDir, uuid, "content.rtf")
	if data, err := os.ReadFile(contentPath); err == nil {
		return rtf.RTFToMarkdown(string(data)), nil
	}

	// Try plain text
	contentPath = filepath.Join(r.filesDir, uuid, "content.txt")
	if data, err := os.ReadFile(contentPath); err == nil {
		return string(data), nil
	}

	// Try older format: Files/Data/{UUID}.rtf
	contentPath = filepath.Join(r.filesDir, uuid+".rtf")
	if data, err := os.ReadFile(contentPath); err == nil {
		return rtf.RTFToMarkdown(string(data)), nil
	}

	// Try older format: Files/Data/{UUID}.txt
	contentPath = filepath.Join(r.filesDir, uuid+".txt")
	if data, err := os.ReadFile(contentPath); err == nil {
		return string(data), nil
	}

	return "", fmt.Errorf("content not found for UUID %s", uuid)
}

// getModificationTime returns the modification time of a document file.
func (r *Reader) getModificationTime(uuid string) time.Time {
	// Try new format
	contentPath := filepath.Join(r.filesDir, uuid, "content.rtf")
	if info, err := os.Stat(contentPath); err == nil {
		return info.ModTime()
	}

	// Try old format
	contentPath = filepath.Join(r.filesDir, uuid+".rtf")
	if info, err := os.Stat(contentPath); err == nil {
		return info.ModTime()
	}

	return time.Now()
}
