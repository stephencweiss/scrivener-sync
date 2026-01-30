package sync

import (
	"fmt"

	"github.com/sweiss/harcroft/internal/config"
)

// RunRemoveAlias removes a project alias from the configuration.
func RunRemoveAlias(alias string) error {
	// 1. Load global config
	globalCfg, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// 2. Remove project
	if err := globalCfg.RemoveProject(alias); err != nil {
		return err
	}

	// 3. Save global config
	if err := globalCfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Project '%s' removed successfully.\n", alias)
	return nil
}
