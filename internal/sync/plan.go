package sync

import (
	"fmt"
	"strings"
	"time"
)

// Plan represents a set of sync operations to be executed.
type Plan struct {
	ToCreateInScriv    []FileChange
	ToCreateInMarkdown []FileChange
	ToUpdateInScriv    []FileChange
	ToUpdateInMarkdown []FileChange
	Conflicts          []Conflict
	Orphans            []Orphan
}

// FileChange represents a single file change operation.
type FileChange struct {
	MarkdownPath string
	ScrivUUID    string
	Title        string
	Content      string
}

// Conflict represents a file that has been modified on both sides.
type Conflict struct {
	MarkdownPath     string
	ScrivUUID        string
	Title            string
	MarkdownContent  string
	ScrivenerContent string
}

// Orphan represents a file that exists on one side but not the other.
type Orphan struct {
	Path         string
	Location     string // "scrivener" or "markdown"
	ScrivUUID    string
	Title        string
	LastSyncTime time.Time
}

// NewPlan creates a new empty sync plan.
func NewPlan() *Plan {
	return &Plan{
		ToCreateInScriv:    []FileChange{},
		ToCreateInMarkdown: []FileChange{},
		ToUpdateInScriv:    []FileChange{},
		ToUpdateInMarkdown: []FileChange{},
		Conflicts:          []Conflict{},
		Orphans:            []Orphan{},
	}
}

// IsEmpty returns true if the plan has no operations.
func (p *Plan) IsEmpty() bool {
	return len(p.ToCreateInScriv) == 0 &&
		len(p.ToCreateInMarkdown) == 0 &&
		len(p.ToUpdateInScriv) == 0 &&
		len(p.ToUpdateInMarkdown) == 0 &&
		len(p.Conflicts) == 0 &&
		len(p.Orphans) == 0
}

// Summary returns a brief summary of the plan.
func (p *Plan) Summary() string {
	var parts []string

	if len(p.ToCreateInScriv) > 0 {
		parts = append(parts, fmt.Sprintf("%d to create in Scrivener", len(p.ToCreateInScriv)))
	}
	if len(p.ToCreateInMarkdown) > 0 {
		parts = append(parts, fmt.Sprintf("%d to create in markdown", len(p.ToCreateInMarkdown)))
	}
	if len(p.ToUpdateInScriv) > 0 {
		parts = append(parts, fmt.Sprintf("%d to update in Scrivener", len(p.ToUpdateInScriv)))
	}
	if len(p.ToUpdateInMarkdown) > 0 {
		parts = append(parts, fmt.Sprintf("%d to update in markdown", len(p.ToUpdateInMarkdown)))
	}
	if len(p.Conflicts) > 0 {
		parts = append(parts, fmt.Sprintf("%d conflicts", len(p.Conflicts)))
	}
	if len(p.Orphans) > 0 {
		parts = append(parts, fmt.Sprintf("%d orphans", len(p.Orphans)))
	}

	if len(parts) == 0 {
		return "No changes to sync"
	}

	return strings.Join(parts, ", ")
}

// PrintStatus prints a detailed status of the plan to stdout.
func (p *Plan) PrintStatus() {
	if p.IsEmpty() {
		fmt.Println("Everything is in sync!")
		return
	}

	fmt.Println("Sync Status")
	fmt.Println(strings.Repeat("=", 50))

	if len(p.ToCreateInScriv) > 0 {
		fmt.Println("\nNew files to create in Scrivener:")
		for _, fc := range p.ToCreateInScriv {
			fmt.Printf("  + %s\n", fc.MarkdownPath)
		}
	}

	if len(p.ToCreateInMarkdown) > 0 {
		fmt.Println("\nNew files to create in markdown:")
		for _, fc := range p.ToCreateInMarkdown {
			fmt.Printf("  + %s (%s)\n", fc.Title, fc.ScrivUUID)
		}
	}

	if len(p.ToUpdateInScriv) > 0 {
		fmt.Println("\nFiles to update in Scrivener (markdown -> Scrivener):")
		for _, fc := range p.ToUpdateInScriv {
			fmt.Printf("  ~ %s\n", fc.MarkdownPath)
		}
	}

	if len(p.ToUpdateInMarkdown) > 0 {
		fmt.Println("\nFiles to update in markdown (Scrivener -> markdown):")
		for _, fc := range p.ToUpdateInMarkdown {
			fmt.Printf("  ~ %s\n", fc.MarkdownPath)
		}
	}

	if len(p.Conflicts) > 0 {
		fmt.Println("\nConflicts (both sides modified):")
		for _, c := range p.Conflicts {
			fmt.Printf("  ! %s (UUID: %s)\n", c.MarkdownPath, c.ScrivUUID)
		}
	}

	if len(p.Orphans) > 0 {
		fmt.Println("\nOrphans (deleted from one side):")
		for _, o := range p.Orphans {
			if o.Location == "markdown" {
				fmt.Printf("  ? %s (deleted from Scrivener)\n", o.Path)
			} else {
				fmt.Printf("  ? %s (deleted from markdown)\n", o.Title)
			}
		}
	}

	fmt.Println()
	fmt.Println(p.Summary())
}

// TotalOperations returns the total number of operations in the plan.
func (p *Plan) TotalOperations() int {
	return len(p.ToCreateInScriv) +
		len(p.ToCreateInMarkdown) +
		len(p.ToUpdateInScriv) +
		len(p.ToUpdateInMarkdown) +
		len(p.Conflicts) +
		len(p.Orphans)
}

// AddCreateInScriv adds a file to be created in Scrivener.
func (p *Plan) AddCreateInScriv(mdPath, title, content string) {
	p.ToCreateInScriv = append(p.ToCreateInScriv, FileChange{
		MarkdownPath: mdPath,
		Title:        title,
		Content:      content,
	})
}

// AddCreateInMarkdown adds a file to be created in markdown.
func (p *Plan) AddCreateInMarkdown(mdPath, scrivUUID, title, content string) {
	p.ToCreateInMarkdown = append(p.ToCreateInMarkdown, FileChange{
		MarkdownPath: mdPath,
		ScrivUUID:    scrivUUID,
		Title:        title,
		Content:      content,
	})
}

// AddUpdateInScriv adds a file to be updated in Scrivener.
func (p *Plan) AddUpdateInScriv(mdPath, scrivUUID, title, content string) {
	p.ToUpdateInScriv = append(p.ToUpdateInScriv, FileChange{
		MarkdownPath: mdPath,
		ScrivUUID:    scrivUUID,
		Title:        title,
		Content:      content,
	})
}

// AddUpdateInMarkdown adds a file to be updated in markdown.
func (p *Plan) AddUpdateInMarkdown(mdPath, scrivUUID, title, content string) {
	p.ToUpdateInMarkdown = append(p.ToUpdateInMarkdown, FileChange{
		MarkdownPath: mdPath,
		ScrivUUID:    scrivUUID,
		Title:        title,
		Content:      content,
	})
}

// AddConflict adds a conflict to the plan.
func (p *Plan) AddConflict(mdPath, scrivUUID, title, mdContent, scrivContent string) {
	p.Conflicts = append(p.Conflicts, Conflict{
		MarkdownPath:     mdPath,
		ScrivUUID:        scrivUUID,
		Title:            title,
		MarkdownContent:  mdContent,
		ScrivenerContent: scrivContent,
	})
}

// AddOrphan adds an orphan to the plan.
func (p *Plan) AddOrphan(path, location, scrivUUID, title string, lastSync time.Time) {
	p.Orphans = append(p.Orphans, Orphan{
		Path:         path,
		Location:     location,
		ScrivUUID:    scrivUUID,
		Title:        title,
		LastSyncTime: lastSync,
	})
}
