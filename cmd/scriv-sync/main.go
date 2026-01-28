package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sweiss/harcroft/internal/config"
	"github.com/sweiss/harcroft/internal/sync"
)

var (
	// Flags for init command
	localPath string
	scrivPath string
	alias     string

	// Global flags
	dryRun         bool
	nonInteractive bool
	version        = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "scriv-sync",
	Short:   "Bi-directional sync between Scrivener and markdown",
	Long:    `A tool for syncing content between Scrivener projects (.scriv) and markdown files.`,
	Version: version,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new sync project",
	Long: `Initialize a new sync project by scanning a Scrivener project
and local directories, then creating a configuration entry.

Example:
  scriv-sync init --local /path/to/markdown --scriv /path/to/Project.scriv --alias myproject`,
	RunE: runInit,
}

var syncCmd = &cobra.Command{
	Use:   "sync <alias>",
	Short: "Bi-directional sync for a project",
	Long: `Perform bi-directional sync between markdown and Scrivener.
Changes on either side are detected and synced.

Example:
  scriv-sync sync myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runSync,
}

var pullCmd = &cobra.Command{
	Use:   "pull <alias>",
	Short: "Sync Scrivener to markdown (Scrivener wins)",
	Long: `Pull changes from Scrivener to markdown files.
Scrivener content takes precedence in conflicts.

Example:
  scriv-sync pull myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runPull,
}

var pushCmd = &cobra.Command{
	Use:   "push <alias>",
	Short: "Sync markdown to Scrivener (markdown wins)",
	Long: `Push changes from markdown to Scrivener.
Markdown content takes precedence in conflicts.

Example:
  scriv-sync push myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runPush,
}

var statusCmd = &cobra.Command{
	Use:   "status <alias>",
	Short: "Show pending changes without syncing",
	Long: `Show the current sync status for a project.
Lists files that would be created, updated, or are in conflict.

Example:
  scriv-sync status myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runStatus,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured projects",
	Long: `List all projects configured in ~/.scriv-sync/config.yaml.

Example:
  scriv-sync list`,
	RunE: runList,
}

func init() {
	// Init command flags
	initCmd.Flags().StringVar(&localPath, "local", "", "path to local markdown directory (required)")
	initCmd.Flags().StringVar(&scrivPath, "scriv", "", "path to Scrivener .scriv project (required)")
	initCmd.Flags().StringVar(&alias, "alias", "", "alias name for this project (required)")
	initCmd.MarkFlagRequired("local")
	initCmd.MarkFlagRequired("scriv")
	initCmd.MarkFlagRequired("alias")

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "skip prompts, use config defaults")

	rootCmd.AddCommand(initCmd, syncCmd, pullCmd, pushCmd, statusCmd, listCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	interactive := !nonInteractive
	return sync.RunInit(alias, localPath, scrivPath, interactive)
}

func runSync(cmd *cobra.Command, args []string) error {
	projectAlias := args[0]

	syncer, err := sync.NewSyncerForAlias(projectAlias)
	if err != nil {
		return err
	}

	interactive := !nonInteractive
	return syncer.Sync(dryRun, interactive)
}

func runPull(cmd *cobra.Command, args []string) error {
	projectAlias := args[0]

	syncer, err := sync.NewSyncerForAlias(projectAlias)
	if err != nil {
		return err
	}

	interactive := !nonInteractive
	return syncer.Pull(dryRun, interactive)
}

func runPush(cmd *cobra.Command, args []string) error {
	projectAlias := args[0]

	syncer, err := sync.NewSyncerForAlias(projectAlias)
	if err != nil {
		return err
	}

	interactive := !nonInteractive
	return syncer.Push(dryRun, interactive)
}

func runStatus(cmd *cobra.Command, args []string) error {
	projectAlias := args[0]

	syncer, err := sync.NewSyncerForAlias(projectAlias)
	if err != nil {
		return err
	}

	return syncer.Status()
}

func runList(cmd *cobra.Command, args []string) error {
	globalCfg, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	aliases := globalCfg.ListProjects()
	if len(aliases) == 0 {
		fmt.Println("No projects configured.")
		fmt.Println("\nTo add a project, run:")
		fmt.Println("  scriv-sync init --local <path> --scriv <path> --alias <name>")
		return nil
	}

	fmt.Println("Configured projects:")
	for _, a := range aliases {
		proj, _ := globalCfg.GetProject(a)
		fmt.Printf("  %s\n", a)
		fmt.Printf("    Local:     %s\n", proj.LocalPath)
		fmt.Printf("    Scrivener: %s\n", proj.ScrivPath)
		enabledCount := len(proj.EnabledMappings())
		fmt.Printf("    Mappings:  %d enabled\n", enabledCount)
	}

	return nil
}
