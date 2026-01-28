package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sweiss/harcroft/internal/config"
	"github.com/sweiss/harcroft/internal/sync"
)

var (
	configPath     string
	dryRun         bool
	nonInteractive bool
	version        = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "scriv-sync",
	Short:   "Bi-directional sync between Scrivener and markdown",
	Long:    `A tool for syncing content between Scrivener projects (.scriv) and markdown files.`,
	Version: version,
	RunE:    runSync,
}

var initCmd = &cobra.Command{
	Use:   "init [path-to-.scriv]",
	Short: "Initialize sync configuration (interactive folder discovery)",
	Long: `Initialize a new sync configuration by scanning a Scrivener project
and local directories, then creating a .scrivener-sync.yaml config file.`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Bi-directional sync (same as running without subcommand)",
	RunE:  runSync,
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Sync Scrivener to markdown (Scrivener wins)",
	RunE:  runPull,
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Sync markdown to Scrivener (markdown wins)",
	RunE:  runPush,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show pending changes without syncing",
	RunE:  runStatus,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", ".scrivener-sync.yaml", "config file path")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "skip prompts, use config defaults")

	rootCmd.AddCommand(initCmd, syncCmd, pullCmd, pushCmd, statusCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	scrivPath := args[0]
	interactive := !nonInteractive

	return sync.RunInit(scrivPath, configPath, interactive)
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	syncer, err := sync.NewSyncer(cfg)
	if err != nil {
		return err
	}

	interactive := !nonInteractive
	return syncer.Sync(dryRun, interactive)
}

func runPull(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	syncer, err := sync.NewSyncer(cfg)
	if err != nil {
		return err
	}

	interactive := !nonInteractive
	return syncer.Pull(dryRun, interactive)
}

func runPush(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	syncer, err := sync.NewSyncer(cfg)
	if err != nil {
		return err
	}

	interactive := !nonInteractive
	return syncer.Push(dryRun, interactive)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	syncer, err := sync.NewSyncer(cfg)
	if err != nil {
		return err
	}

	return syncer.Status()
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s\nRun 'scriv-sync init <path-to-.scriv>' to create one", configPath)
		}
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if errs := cfg.Validate(); len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "Config validation errors:")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		return nil, fmt.Errorf("invalid configuration")
	}

	return cfg, nil
}
