package sync

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sweiss/harcroft/internal/config"
	"github.com/sweiss/harcroft/internal/scrivener"
)

// Syncer handles bi-directional sync between markdown and Scrivener.
type Syncer struct {
	config *config.ProjectConfig
	state  *State
	reader *scrivener.Reader
	writer *scrivener.Writer

	mdRoot    string
	scrivPath string
	alias     string
}

// NewSyncerForAlias creates a new Syncer for the given project alias.
func NewSyncerForAlias(alias string) (*Syncer, error) {
	globalCfg, err := config.LoadGlobal()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	projCfg, err := globalCfg.GetProject(alias)
	if err != nil {
		return nil, err
	}

	return NewSyncer(projCfg, alias)
}

// NewSyncer creates a new Syncer from the given project configuration.
func NewSyncer(cfg *config.ProjectConfig, alias string) (*Syncer, error) {
	scrivPath, err := cfg.ScrivenerPath()
	if err != nil {
		return nil, err
	}

	mdRoot := cfg.MarkdownPath()

	reader, err := scrivener.NewReader(scrivPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Scrivener project for reading: %w", err)
	}

	writer, err := scrivener.NewWriter(scrivPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Scrivener project for writing: %w", err)
	}

	state, err := LoadStateForAlias(alias)
	if err != nil {
		return nil, fmt.Errorf("failed to load sync state: %w", err)
	}
	state.SetScrivPath(scrivPath)

	return &Syncer{
		config:    cfg,
		state:     state,
		reader:    reader,
		writer:    writer,
		mdRoot:    mdRoot,
		scrivPath: scrivPath,
		alias:     alias,
	}, nil
}

// Sync performs bi-directional sync.
func (s *Syncer) Sync(dryRun, interactive bool) error {
	plan, err := s.detectAllChanges()
	if err != nil {
		return err
	}

	if plan.IsEmpty() {
		fmt.Println("Everything is in sync!")
		return nil
	}

	plan.PrintStatus()

	if dryRun {
		fmt.Println("\n(dry-run mode - no changes applied)")
		return nil
	}

	return s.executePlan(plan, interactive)
}

// Pull syncs from Scrivener to markdown.
func (s *Syncer) Pull(dryRun, interactive bool) error {
	plan, err := s.detectAllChanges()
	if err != nil {
		return err
	}

	// Filter plan to only Scrivener -> markdown changes
	pullPlan := NewPlan()
	pullPlan.ToCreateInMarkdown = plan.ToCreateInMarkdown
	pullPlan.ToUpdateInMarkdown = plan.ToUpdateInMarkdown
	// Include orphans that exist in markdown but not Scrivener
	for _, o := range plan.Orphans {
		if o.Location == "markdown" {
			pullPlan.Orphans = append(pullPlan.Orphans, o)
		}
	}

	if pullPlan.IsEmpty() {
		fmt.Println("No changes to pull from Scrivener.")
		return nil
	}

	pullPlan.PrintStatus()

	if dryRun {
		fmt.Println("\n(dry-run mode - no changes applied)")
		return nil
	}

	return s.executePlan(pullPlan, interactive)
}

// Push syncs from markdown to Scrivener.
func (s *Syncer) Push(dryRun, interactive bool) error {
	plan, err := s.detectAllChanges()
	if err != nil {
		return err
	}

	// Filter plan to only markdown -> Scrivener changes
	pushPlan := NewPlan()
	pushPlan.ToCreateInScriv = plan.ToCreateInScriv
	pushPlan.ToUpdateInScriv = plan.ToUpdateInScriv
	// Include orphans that exist in Scrivener but not markdown
	for _, o := range plan.Orphans {
		if o.Location == "scrivener" {
			pushPlan.Orphans = append(pushPlan.Orphans, o)
		}
	}

	if pushPlan.IsEmpty() {
		fmt.Println("No changes to push to Scrivener.")
		return nil
	}

	pushPlan.PrintStatus()

	if dryRun {
		fmt.Println("\n(dry-run mode - no changes applied)")
		return nil
	}

	return s.executePlan(pushPlan, interactive)
}

// Status shows the current sync status without making changes.
func (s *Syncer) Status() error {
	plan, err := s.detectAllChanges()
	if err != nil {
		return err
	}

	plan.PrintStatus()
	return nil
}

// detectAllChanges scans both sides and creates a sync plan.
func (s *Syncer) detectAllChanges() (*Plan, error) {
	plan := NewPlan()

	for _, mapping := range s.config.EnabledMappings() {
		if err := s.detectChangesForMapping(mapping, plan); err != nil {
			return nil, err
		}
	}

	// Detect orphans (files that were synced before but now missing from one side)
	s.detectOrphans(plan)

	return plan, nil
}

// detectChangesForMapping detects changes for a single folder mapping.
func (s *Syncer) detectChangesForMapping(mapping config.FolderMapping, plan *Plan) error {
	mdDir := filepath.Join(s.mdRoot, mapping.MarkdownDir)

	// Get Scrivener folder
	scrivFolder, err := s.reader.FindFolderByTitle(mapping.ScrivenerFolder)
	if err != nil {
		// Folder doesn't exist in Scrivener
		if s.config.Options.CreateMissingFolders {
			// Will create when syncing
		} else {
			return fmt.Errorf("Scrivener folder '%s' not found", mapping.ScrivenerFolder)
		}
	}

	// Get markdown files
	mdFiles, err := s.getMarkdownFiles(mdDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Get Scrivener documents
	var scrivDocs []*scrivener.Document
	if scrivFolder != nil {
		scrivDocs = scrivFolder.Children
	}

	// Build lookup maps
	mdFileMap := make(map[string]string) // title -> path
	for _, path := range mdFiles {
		title := titleFromFilename(filepath.Base(path))
		mdFileMap[strings.ToLower(title)] = path
	}

	scrivDocMap := make(map[string]*scrivener.Document) // title -> doc
	for _, doc := range scrivDocs {
		if !doc.IsFolder() {
			scrivDocMap[strings.ToLower(doc.Title)] = doc
		}
	}

	// Check each markdown file
	for _, mdPath := range mdFiles {
		title := titleFromFilename(filepath.Base(mdPath))
		lowerTitle := strings.ToLower(title)

		mdContent, err := os.ReadFile(mdPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", mdPath, err)
		}
		mdHash := computeHash(string(mdContent))

		scrivDoc := scrivDocMap[lowerTitle]
		if scrivDoc == nil {
			// Markdown file exists, Scrivener doc doesn't
			if !s.state.WasPreviouslySynced(mdPath) {
				plan.AddCreateInScriv(mdPath, title, string(mdContent))
			}
			// If was previously synced, it will be handled as orphan
		} else {
			// Both exist - check for changes
			scrivHash := scrivDoc.ContentHash()
			conflict := s.state.DetectConflict(mdPath, mdHash, scrivDoc.UUID, scrivHash)

			switch conflict {
			case ConflictNewFile:
				// New file on both sides with same title - treat as conflict
				plan.AddConflict(mdPath, scrivDoc.UUID, title, string(mdContent), scrivDoc.Content)
			case ConflictMarkdownOnly:
				plan.AddUpdateInScriv(mdPath, scrivDoc.UUID, title, string(mdContent))
			case ConflictScrivenerOnly:
				plan.AddUpdateInMarkdown(mdPath, scrivDoc.UUID, title, scrivDoc.Content)
			case ConflictBoth:
				plan.AddConflict(mdPath, scrivDoc.UUID, title, string(mdContent), scrivDoc.Content)
			case ConflictNone:
				// No changes needed
			}

			delete(scrivDocMap, lowerTitle)
		}
	}

	// Remaining Scrivener docs don't have matching markdown files
	for _, doc := range scrivDocMap {
		if doc.IsFolder() {
			continue
		}
		mdPath := filepath.Join(mdDir, sanitizeFilename(doc.Title)+".md")
		if !s.state.WasPreviouslySynced(mdPath) {
			plan.AddCreateInMarkdown(mdPath, doc.UUID, doc.Title, doc.Content)
		}
		// If was previously synced, it will be handled as orphan
	}

	return nil
}

// detectOrphans finds files that were previously synced but now exist only on one side.
func (s *Syncer) detectOrphans(plan *Plan) {
	for _, mdPath := range s.state.AllTrackedPaths() {
		// Check if markdown file still exists
		mdExists := fileExists(mdPath)

		// Check if Scrivener doc still exists
		uuid := s.state.GetUUIDForPath(mdPath)
		scrivExists := s.scrivDocExists(uuid)

		if mdExists && !scrivExists {
			// Markdown exists, Scrivener deleted
			fs := s.state.GetFileState(mdPath)
			var lastSync time.Time
			if fs != nil {
				lastSync, _ = time.Parse(time.RFC3339, fs.LastSynced)
			}
			plan.AddOrphan(mdPath, "markdown", uuid, titleFromFilename(filepath.Base(mdPath)), lastSync)
		} else if !mdExists && scrivExists {
			// Markdown deleted, Scrivener exists
			fs := s.state.GetFileState(mdPath)
			var lastSync time.Time
			var title string
			if fs != nil {
				lastSync, _ = time.Parse(time.RFC3339, fs.LastSynced)
				title = titleFromFilename(filepath.Base(mdPath))
			}
			plan.AddOrphan(mdPath, "scrivener", uuid, title, lastSync)
		} else if !mdExists && !scrivExists {
			// Both deleted - just clean up state
			s.state.RemoveFile(mdPath)
		}
	}
}

// scrivDocExists checks if a Scrivener document with the given UUID exists.
func (s *Syncer) scrivDocExists(uuid string) bool {
	if uuid == "" {
		return false
	}
	docs, err := s.reader.GetAllDocuments()
	if err != nil {
		return false
	}
	for _, doc := range docs {
		if doc.UUID == uuid {
			return true
		}
	}
	return false
}

// executePlan executes the sync plan.
func (s *Syncer) executePlan(plan *Plan, interactive bool) error {
	// Handle conflicts first
	for _, conflict := range plan.Conflicts {
		resolution, err := s.resolveConflict(conflict, interactive)
		if err != nil {
			return err
		}

		switch resolution {
		case "markdown":
			// Use markdown content
			if err := s.writer.UpdateDocumentContent(conflict.ScrivUUID, conflict.MarkdownContent, true); err != nil {
				return err
			}
			s.recordSync(conflict.MarkdownPath, conflict.ScrivUUID, conflict.MarkdownContent)
		case "scrivener":
			// Use Scrivener content
			if err := os.WriteFile(conflict.MarkdownPath, []byte(conflict.ScrivenerContent), 0644); err != nil {
				return err
			}
			s.recordSync(conflict.MarkdownPath, conflict.ScrivUUID, conflict.ScrivenerContent)
		case "skip":
			fmt.Printf("  Skipped conflict: %s\n", conflict.MarkdownPath)
		}
	}

	// Create in Scrivener
	for _, fc := range plan.ToCreateInScriv {
		fmt.Printf("  Creating in Scrivener: %s\n", fc.Title)

		// Find or create parent folder
		folderUUID, err := s.ensureScrivenerFolder(fc.MarkdownPath)
		if err != nil {
			return err
		}

		uuid, err := s.writer.CreateDocument(fc.Title, fc.Content, folderUUID, true)
		if err != nil {
			return fmt.Errorf("failed to create document '%s': %w", fc.Title, err)
		}

		s.recordSync(fc.MarkdownPath, uuid, fc.Content)
	}

	// Create in markdown
	for _, fc := range plan.ToCreateInMarkdown {
		fmt.Printf("  Creating in markdown: %s\n", fc.MarkdownPath)

		// Ensure directory exists
		dir := filepath.Dir(fc.MarkdownPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := os.WriteFile(fc.MarkdownPath, []byte(fc.Content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", fc.MarkdownPath, err)
		}

		s.recordSync(fc.MarkdownPath, fc.ScrivUUID, fc.Content)
	}

	// Update in Scrivener
	for _, fc := range plan.ToUpdateInScriv {
		fmt.Printf("  Updating in Scrivener: %s\n", fc.Title)

		if err := s.writer.UpdateDocumentContent(fc.ScrivUUID, fc.Content, true); err != nil {
			return fmt.Errorf("failed to update document '%s': %w", fc.Title, err)
		}

		s.recordSync(fc.MarkdownPath, fc.ScrivUUID, fc.Content)
	}

	// Update in markdown
	for _, fc := range plan.ToUpdateInMarkdown {
		fmt.Printf("  Updating in markdown: %s\n", fc.MarkdownPath)

		if err := os.WriteFile(fc.MarkdownPath, []byte(fc.Content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", fc.MarkdownPath, err)
		}

		s.recordSync(fc.MarkdownPath, fc.ScrivUUID, fc.Content)
	}

	// Handle orphans
	orphanActions := make(map[string]DeletionAction)
	for _, orphan := range plan.Orphans {
		action := resolveOrphanAction(orphan, s.config.Options.DefaultDeletionAction, interactive)

		key := orphan.Path
		if orphan.Location == "scrivener" {
			key = orphan.ScrivUUID
		}
		orphanActions[key] = action

		if err := s.executeOrphanAction(orphan, action); err != nil {
			return err
		}
	}

	// Save Scrivener changes
	if err := s.writer.Save(); err != nil {
		return fmt.Errorf("failed to save Scrivener project: %w", err)
	}

	// Save state
	s.state.UpdateLastSync()
	if err := s.state.Save(); err != nil {
		return fmt.Errorf("failed to save sync state: %w", err)
	}

	fmt.Println("\nSync completed successfully!")
	return nil
}

// resolveConflict prompts the user to resolve a conflict.
func (s *Syncer) resolveConflict(conflict Conflict, interactive bool) (string, error) {
	if !interactive {
		return s.config.Options.DefaultConflictResolution, nil
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Printf("Conflict detected: %s\n", conflict.MarkdownPath)
	fmt.Println("  Both the markdown file and Scrivener document have been modified.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  [m] Use markdown version (overwrite Scrivener)")
	fmt.Println("  [s] Use Scrivener version (overwrite markdown)")
	fmt.Println("  [k] Skip (leave both as-is for now)")

	for {
		fmt.Print("\nChoice: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "skip", nil
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "m", "markdown":
			return "markdown", nil
		case "s", "scrivener":
			return "scrivener", nil
		case "k", "skip":
			return "skip", nil
		default:
			fmt.Println("Invalid choice. Please enter m, s, or k.")
		}
	}
}

// executeOrphanAction handles an orphan based on the chosen action.
func (s *Syncer) executeOrphanAction(orphan Orphan, action DeletionAction) error {
	switch action {
	case ActionDelete:
		if orphan.Location == "markdown" {
			// Delete the markdown file
			fmt.Printf("  Deleting markdown file: %s\n", orphan.Path)
			if err := os.Remove(orphan.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete %s: %w", orphan.Path, err)
			}
			s.state.RemoveFile(orphan.Path)
		} else {
			// Delete from Scrivener - this is more complex and might need additional implementation
			fmt.Printf("  Note: Deleting from Scrivener not yet implemented. Skipping: %s\n", orphan.Title)
		}

	case ActionRecreate:
		if orphan.Location == "markdown" {
			// Recreate in Scrivener from markdown
			content, err := os.ReadFile(orphan.Path)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", orphan.Path, err)
			}

			folderUUID, err := s.ensureScrivenerFolder(orphan.Path)
			if err != nil {
				return err
			}

			uuid, err := s.writer.CreateDocument(orphan.Title, string(content), folderUUID, true)
			if err != nil {
				return fmt.Errorf("failed to recreate document '%s': %w", orphan.Title, err)
			}

			fmt.Printf("  Recreated in Scrivener: %s\n", orphan.Title)
			s.recordSync(orphan.Path, uuid, string(content))
		} else {
			// Recreate markdown from Scrivener
			docs, _ := s.reader.GetAllDocuments()
			for _, doc := range docs {
				if doc.UUID == orphan.ScrivUUID {
					if err := os.WriteFile(orphan.Path, []byte(doc.Content), 0644); err != nil {
						return fmt.Errorf("failed to recreate %s: %w", orphan.Path, err)
					}
					fmt.Printf("  Recreated markdown: %s\n", orphan.Path)
					s.recordSync(orphan.Path, orphan.ScrivUUID, doc.Content)
					break
				}
			}
		}

	case ActionSkip:
		fmt.Printf("  Skipped orphan: %s\n", orphan.Path)
	}

	return nil
}

// ensureScrivenerFolder finds or creates the Scrivener folder for a markdown path.
func (s *Syncer) ensureScrivenerFolder(mdPath string) (string, error) {
	// Determine which mapping this path belongs to
	relPath, err := filepath.Rel(s.mdRoot, mdPath)
	if err != nil {
		return "", err
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) < 2 {
		return "", nil // Root level, no parent folder
	}

	mdDir := parts[0]

	// Find the mapping
	for _, mapping := range s.config.EnabledMappings() {
		if mapping.MarkdownDir == mdDir {
			uuid, err := s.writer.FindFolderByTitle(mapping.ScrivenerFolder)
			if err != nil {
				// Create the folder
				if s.config.Options.CreateMissingFolders {
					return s.writer.CreateFolder(mapping.ScrivenerFolder, "")
				}
				return "", fmt.Errorf("Scrivener folder '%s' not found", mapping.ScrivenerFolder)
			}
			return uuid, nil
		}
	}

	return "", nil
}

// recordSync records a successful sync in the state.
func (s *Syncer) recordSync(mdPath, scrivUUID, content string) {
	hash := computeHash(content)
	s.state.RecordFile(mdPath, scrivUUID, hash, time.Now())
}

// getMarkdownFiles returns all .md files in a directory.
func (s *Syncer) getMarkdownFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// computeHash returns the MD5 hash of a string.
func computeHash(content string) string {
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}
