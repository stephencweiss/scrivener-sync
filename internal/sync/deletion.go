package sync

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// DeletionAction represents how to handle an orphaned file.
type DeletionAction string

const (
	// ActionDelete deletes the orphan from the remaining side.
	ActionDelete DeletionAction = "delete"
	// ActionRecreate recreates the orphan on the missing side.
	ActionRecreate DeletionAction = "recreate"
	// ActionSkip skips handling the orphan.
	ActionSkip DeletionAction = "skip"
)

// promptDeletionAction prompts the user for how to handle an orphan.
func promptDeletionAction(orphan Orphan, defaultAction string) DeletionAction {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	if orphan.Location == "markdown" {
		fmt.Printf("Orphan detected: '%s'\n", orphan.Path)
		fmt.Println("  This markdown file exists but the corresponding Scrivener document was deleted.")
	} else {
		fmt.Printf("Orphan detected: '%s' (UUID: %s)\n", orphan.Title, orphan.ScrivUUID)
		fmt.Println("  This Scrivener document exists but the corresponding markdown file was deleted.")
	}

	if !orphan.LastSyncTime.IsZero() {
		fmt.Printf("  Last synced: %s\n", orphan.LastSyncTime.Format("2006-01-02 15:04:05"))
	}

	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  [d] Delete - Remove from the remaining side")
	fmt.Println("  [r] Recreate - Restore on the missing side")
	fmt.Println("  [s] Skip - Leave as is for now")

	defaultKey := "s"
	switch defaultAction {
	case "delete":
		defaultKey = "d"
	case "recreate":
		defaultKey = "r"
	case "skip":
		defaultKey = "s"
	}

	for {
		fmt.Printf("\nChoice [%s]: ", defaultKey)
		input, err := reader.ReadString('\n')
		if err != nil {
			return ActionSkip
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" {
			input = defaultKey
		}

		switch input {
		case "d", "delete":
			return ActionDelete
		case "r", "recreate":
			return ActionRecreate
		case "s", "skip":
			return ActionSkip
		default:
			fmt.Println("Invalid choice. Please enter d, r, or s.")
		}
	}
}

// resolveOrphanAction determines the action for an orphan based on config or prompt.
func resolveOrphanAction(orphan Orphan, defaultAction string, interactive bool) DeletionAction {
	if !interactive {
		switch defaultAction {
		case "delete":
			return ActionDelete
		case "recreate":
			return ActionRecreate
		case "skip":
			return ActionSkip
		default:
			return ActionSkip
		}
	}

	return promptDeletionAction(orphan, defaultAction)
}

// formatOrphanSummary returns a summary string for orphan actions.
func formatOrphanSummary(orphans []Orphan, actions map[string]DeletionAction) string {
	if len(orphans) == 0 {
		return "No orphans found"
	}

	deleteCount := 0
	recreateCount := 0
	skipCount := 0

	for _, o := range orphans {
		key := o.Path
		if o.Location == "scrivener" {
			key = o.ScrivUUID
		}
		switch actions[key] {
		case ActionDelete:
			deleteCount++
		case ActionRecreate:
			recreateCount++
		case ActionSkip:
			skipCount++
		}
	}

	var parts []string
	if deleteCount > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", deleteCount))
	}
	if recreateCount > 0 {
		parts = append(parts, fmt.Sprintf("%d recreated", recreateCount))
	}
	if skipCount > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipCount))
	}

	return fmt.Sprintf("Orphans: %s", strings.Join(parts, ", "))
}
