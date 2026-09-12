package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/adnanex/dbctl/pkg/compose"
	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/ui"
)

//go:embed longdesc/init.txt
var initLong string

var initCmd = &cobra.Command{
	Use:   "init [destination]",
	Short: "Scaffold a dbctl configuration file",
	Long:  strings.TrimRight(initLong, "\n"),
	Example: `  dbctl init                  # Create ./config.yaml with local template
  dbctl init global           # Create ~/.dbctl/config.yaml with global defaults
  dbctl init minimal          # Create ./config.yaml with minimal zero-admin template
  dbctl init ~/my-config.yaml # Create template at a custom path
  dbctl init global --force   # Overwrite existing global config
  dbctl init stack            # Scaffold a full database stack (./dbs/docker-compose.yml)
  dbctl init stack global     # Scaffold the stack at ~/.dbctl/dbs/docker-compose.yml`,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"scaffold", "create-config"},
	RunE:    runInit,
}

var initStackCmd = &cobra.Command{
	Use:   "stack [destination]",
	Short: "Scaffold a full database stack (Docker Compose) and matching config",
	Example: `  dbctl init stack                     # ./dbs/docker-compose.yml + ./config.yaml
  dbctl init stack global              # ~/.dbctl/dbs/docker-compose.yml + ~/.dbctl/config.yaml
  dbctl init stack --engines mysql,redis
  dbctl init stack global --force`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitStack,
}

func init() {
	initCmd.Flags().StringP("type", "T", "", "template type: 'global', 'local', or 'minimal' (defaults based on destination)")
	initCmd.Flags().BoolP("force", "f", false, "overwrite existing configuration file if it exists")
	initCmd.Flags().Bool("stack", false, "scaffold a full database stack instead of a driver config")
	initCmd.Flags().StringSlice("engines", nil, fmt.Sprintf("engines to include in the stack (default: full stack — %s)", strings.Join(compose.StackEngineKeys(), ", ")))

	initStackCmd.Flags().BoolP("force", "f", false, "overwrite an existing docker-compose.yml (config.yaml is always safely merged, never overwritten by this flag)")
	initStackCmd.Flags().StringSlice("engines", nil, fmt.Sprintf("engines to include (default: full stack — %s)", strings.Join(compose.StackEngineKeys(), ", ")))

	initCmd.AddCommand(initStackCmd)
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if stack, _ := cmd.Flags().GetBool("stack"); stack {
		return runInitStack(cmd, args)
	}

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

func runInitStack(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	engines, _ := cmd.Flags().GetStringSlice("engines")

	dest := "local"
	if len(args) > 0 {
		dest = args[0]
	}

	stackDir, configPath, err := config.ResolveStackPath(dest)
	if err != nil {
		return err
	}

	if len(engines) == 0 {
		engines, err = selectStackEngines()
		if err != nil {
			return err
		}
	}

	composeFilePath, err := compose.WriteStack(stackDir, engines, force)
	if err != nil {
		return err
	}

	if err := compose.WriteStackConfig(configPath, engines, composeFilePath); err != nil {
		return err
	}

	ui.Info("Database stack created",
		"engines", strings.Join(engines, ", "),
		"compose", composeFilePath,
		"config", configPath,
	)

	return nil
}

// selectStackEngines prompts the user to choose engines interactively when
// running in a TTY; otherwise it defaults to the full stack.
func selectStackEngines() ([]string, error) {
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return compose.StackEngineKeys(), nil
	}

	keys := compose.StackEngineKeys()
	options := make([]huh.Option[string], 0, len(keys))
	for _, k := range keys {
		options = append(options, huh.NewOption(k, k).Selected(true))
	}

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title("Select database stack engines").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, fmt.Errorf("engine selection cancelled: %w", err)
	}

	if len(selected) == 0 {
		return keys, nil
	}
	return selected, nil
}
