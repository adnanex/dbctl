package cmd

import (
	"github.com/spf13/cobra"

	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/ui"
)

var initCmd = &cobra.Command{
	Use:   "init [destination]",
	Short: "Scaffold a dbctl configuration file",
	Long: `Generate a template configuration file for dbctl.

Destinations:
  local / .       Create ./config.yaml (default)
  global / ~      Create ~/.dbctl/config.yaml
  config / .config Create ~/.config/dbctl/config.yaml
  minimal         Create a minimal ./config.yaml (no admin credentials needed)
  <filepath>      Create template at a custom filepath`,
	Example: `  dbctl init                  # Create ./config.yaml with local template
  dbctl init global           # Create ~/.dbctl/config.yaml with global defaults
  dbctl init minimal          # Create ./config.yaml with minimal zero-admin template
  dbctl init ~/my-config.yaml # Create template at a custom path
  dbctl init global --force   # Overwrite existing global config`,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"scaffold", "create-config"},
	RunE:    runInit,
}

func init() {
	initCmd.Flags().StringP("type", "T", "", "template type: 'global', 'local', or 'minimal' (defaults based on destination)")
	initCmd.Flags().BoolP("force", "f", false, "overwrite existing configuration file if it exists")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	templateType, _ := cmd.Flags().GetString("type")

	dest := "local"
	if len(args) > 0 {
		dest = args[0]
	}

	targetPath, isGlobal, err := config.ResolveInitPath(dest)
	if err != nil {
		return err
	}

	// Determine template type
	tpl := templateType
	if tpl == "" {
		switch dest {
		case "minimal":
			tpl = "minimal"
		default:
			if isGlobal {
				tpl = "global"
			} else {
				tpl = "local"
			}
		}
	}

	if err := config.GenerateConfigFile(targetPath, tpl, force); err != nil {
		return err
	}

	ui.Info("Configuration created",
		"template", tpl,
		"path", targetPath,
	)

	return nil
}
