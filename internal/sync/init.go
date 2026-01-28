package sync

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sweiss/harcroft/internal/config"
	"github.com/sweiss/harcroft/internal/scrivener"
)

// RunInit runs the initialization process for a new project.
func RunInit(alias, localPath, scrivPath string, interactive bool) error {
	// 1. Load global config
	globalCfg, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// 2. Check if alias already exists
	if globalCfg.HasProject(alias) {
		return fmt.Errorf("project '%s' already exists. Choose a different alias or remove the existing one", alias)
	}

	// 3. Validate paths
	localPath, err = filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("failed to resolve local path: %w", err)
	}

	scrivPath, err = filepath.Abs(scrivPath)
	if err != nil {
		return fmt.Errorf("failed to resolve scriv path: %w", err)
	}

	// Check local path exists
	if info, err := os.Stat(localPath); err != nil || !info.IsDir() {
		return fmt.Errorf("local path does not exist or is not a directory: %s", localPath)
	}

	// 4. Validate Scrivener project
	fmt.Println("Scanning Scrivener project...")
	reader, err := scrivener.NewReader(scrivPath)
	if err != nil {
		return fmt.Errorf("failed to open Scrivener project: %w", err)
	}

	// 5. Get Scrivener folders
	folders, err := reader.GetTopLevelFolders()
	if err != nil {
		return fmt.Errorf("failed to read Scrivener folders: %w", err)
	}

	fmt.Printf("  Found folders: ")
	var folderNames []string
	for _, f := range folders {
		folderNames = append(folderNames, f.Title)
	}
	fmt.Println(strings.Join(folderNames, ", "))

	// 6. Scan local directories
	fmt.Println("\nScanning local directories...")
	localDirs := scanLocalDirectories(localPath)
	if len(localDirs) > 0 {
		fmt.Printf("  Found: %s\n", strings.Join(localDirs, ", "))
	} else {
		fmt.Println("  No directories found")
	}

	// 7. Suggest mappings
	mappings := suggestMappings(folders, localDirs)

	// 8. Interactive selection
	if interactive && len(mappings) > 0 {
		mappings = interactiveMappingSelection(mappings, localPath)
	}

	// 9. Add project to global config
	proj := globalCfg.AddProject(alias, localPath, scrivPath)
	for _, m := range mappings {
		proj.AddMapping(m.MarkdownDir, m.ScrivenerFolder, m.SyncEnabled)
	}

	// 10. Save global config
	if err := globalCfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	enabledCount := len(proj.EnabledMappings())
	configPath, _ := config.ConfigPath()
	fmt.Printf("\nProject '%s' added to %s with %d folder mapping(s).\n", alias, configPath, enabledCount)
	fmt.Printf("\nTo sync, run: scriv-sync sync %s\n", alias)

	return nil
}

// suggestMappings creates suggested folder mappings based on name matching.
func suggestMappings(scrivFolders []*scrivener.Document, localDirs []string) []config.FolderMapping {
	var mappings []config.FolderMapping

	// Create a map of lowercase local dir names for matching
	localDirMap := make(map[string]string)
	for _, dir := range localDirs {
		localDirMap[strings.ToLower(dir)] = dir
	}

	for _, folder := range scrivFolders {
		lowerTitle := strings.ToLower(folder.Title)
		mapping := config.FolderMapping{
			ScrivenerFolder: folder.Title,
			SyncEnabled:     false,
		}

		// Check for exact case-insensitive match
		if localDir, exists := localDirMap[lowerTitle]; exists {
			mapping.MarkdownDir = localDir
			mapping.SyncEnabled = true
		} else {
			// No match - suggest creating directory
			mapping.MarkdownDir = strings.ToLower(folder.Title)
		}

		mappings = append(mappings, mapping)
	}

	return mappings
}

// interactiveMappingSelection allows user to toggle mappings.
func interactiveMappingSelection(mappings []config.FolderMapping, localPath string) []config.FolderMapping {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nSuggested mappings:")
	printMappings(mappings, localPath)

	fmt.Println("\nCommands:")
	fmt.Println("  [1-9] Toggle mapping on/off")
	fmt.Println("  [a]   Accept and continue")
	fmt.Println("  [c]   Create missing directories and accept")
	fmt.Println("  [q]   Quit without saving")

	for {
		fmt.Print("\nChoice: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return mappings
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "a":
			return mappings
		case "c":
			// Create missing directories for enabled mappings
			for _, m := range mappings {
				dirPath := filepath.Join(localPath, m.MarkdownDir)
				if m.SyncEnabled && !directoryExists(dirPath) {
					if err := os.MkdirAll(dirPath, 0755); err != nil {
						fmt.Printf("Warning: failed to create %s: %v\n", dirPath, err)
					} else {
						fmt.Printf("Created directory: %s\n", dirPath)
					}
				}
			}
			return mappings
		case "q":
			fmt.Println("Aborted.")
			os.Exit(0)
		default:
			// Try to parse as number
			var num int
			if _, err := fmt.Sscanf(input, "%d", &num); err == nil {
				if num >= 1 && num <= len(mappings) {
					mappings[num-1].SyncEnabled = !mappings[num-1].SyncEnabled
					printMappings(mappings, localPath)
				} else {
					fmt.Printf("Invalid number. Enter 1-%d.\n", len(mappings))
				}
			} else {
				fmt.Println("Invalid input. Enter a number, 'a', 'c', or 'q'.")
			}
		}
	}
}

// printMappings displays the current mapping selections.
func printMappings(mappings []config.FolderMapping, localPath string) {
	for i, m := range mappings {
		checkmark := " "
		if m.SyncEnabled {
			checkmark = "x"
		}

		dirPath := filepath.Join(localPath, m.MarkdownDir)
		dirStatus := m.MarkdownDir
		if !directoryExists(dirPath) {
			dirStatus = fmt.Sprintf("(create) %s", m.MarkdownDir)
		}

		fmt.Printf("  [%s] %d. %s  <->  %s\n", checkmark, i+1, dirStatus, m.ScrivenerFolder)
	}
}

// scanLocalDirectories finds all directories in the given root.
func scanLocalDirectories(root string) []string {
	var dirs []string

	entries, err := os.ReadDir(root)
	if err != nil {
		return dirs
	}

	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			// Skip hidden directories and common non-content directories
			if strings.HasPrefix(name, ".") ||
				name == "node_modules" ||
				name == "vendor" ||
				name == "plans" ||
				name == "cmd" ||
				name == "internal" ||
				name == "scriv-sync" {
				continue
			}
			dirs = append(dirs, name)
		}
	}

	return dirs
}

// directoryExists checks if a directory exists.
func directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// sanitizeFilename converts a title to a safe filename.
func sanitizeFilename(title string) string {
	// Convert to lowercase
	name := strings.ToLower(title)

	// Replace spaces and special characters
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	name = replacer.Replace(name)

	// Remove multiple consecutive dashes
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Trim leading/trailing dashes
	name = strings.Trim(name, "-")

	return name
}

// titleFromFilename converts a filename back to a title.
func titleFromFilename(filename string) string {
	// Remove .md extension
	name := strings.TrimSuffix(filename, ".md")
	name = strings.TrimSuffix(name, filepath.Ext(name))

	// Replace dashes with spaces
	name = strings.ReplaceAll(name, "-", " ")

	// Title case
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}
