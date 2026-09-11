package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adnanex/dbctl/pkg/compose"
	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/ui"
)

//go:embed longdesc/up.txt
var upLong string

//go:embed longdesc/down.txt
var downLong string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Generate Docker Compose and start database containers",
	Long:  strings.TrimRight(upLong, "\n"),
	Example: `  dbctl up                          # Generate compose + start containers
  dbctl up --generate-only          # Just generate docker-compose.dbctl.yml
  dbctl up --compose ./my-compose.yml  # Use custom compose file
  dbctl up --detach                 # Run in background (default)`,
	RunE: runUp,
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop database containers",
	Long:  strings.TrimRight(downLong, "\n"),
	Example: `  dbctl down                # Stop containers
  dbctl down --volumes      # Stop containers and remove data volumes`,
	RunE: runDown,
}

func init() {
	upCmd.Flags().Bool("generate-only", false, "generate the compose file without starting containers")
	upCmd.Flags().String("compose", "", "use a custom Docker Compose file instead of auto-generating")
	upCmd.Flags().String("output", "docker-compose.dbctl.yml", "output path for generated compose file")
	upCmd.Flags().BoolP("detach", "d", true, "run containers in detached mode")

	downCmd.Flags().BoolP("volumes", "V", false, "also remove data volumes")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	generateOnly, _ := cmd.Flags().GetBool("generate-only")
	customCompose, _ := cmd.Flags().GetString("compose")
	output, _ := cmd.Flags().GetString("output")

	// If using custom compose file, just run it
	if customCompose != "" {
		if _, err := os.Stat(customCompose); os.IsNotExist(err) {
			return fmt.Errorf("compose file not found: %s", customCompose)
		}
		ui.Info("Using custom compose file", "path", customCompose)
		if !generateOnly {
			return compose.Up(customCompose)
		}
		return nil
	}

	// Load config to generate compose
	configFile := cfgFile
	if configFile == "" {
		configFile = "config.yaml"
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s (run 'dbctl init' first)", configFile)
	}

	cfg, err := config.LoadConfig(configFile, nil)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Load global config for admin credentials
	globalPath := config.FindGlobalConfigFile("")
	if globalPath != "" {
		globalCfg, err := config.LoadGlobalConfig(globalPath, nil)
		if err == nil {
			config.MergeGlobalDefaults(cfg, globalCfg)
		}
	}

	if len(cfg.Targets) == 0 {
		return fmt.Errorf("no target databases defined in %s", configFile)
	}

	// Collect unique drivers
	drivers := make([]string, 0)
	seen := make(map[string]bool)
	for _, t := range cfg.Targets {
		if !seen[t.Driver] {
			drivers = append(drivers, t.Driver)
			seen[t.Driver] = true
		}
	}

	ui.Info("Generating Docker Compose file",
		"targets", strings.Join(drivers, ", "),
		"output", output,
	)

	content, err := compose.Generate(cfg)
	if err != nil {
		return fmt.Errorf("error generating compose file: %w", err)
	}

	if err := os.WriteFile(output, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing compose file: %w", err)
	}

	ui.Info(fmt.Sprintf("%s Compose file generated", ui.StatusIcon(true)), "path", output)

	if generateOnly {
		return nil
	}

	ui.Info("Starting containers...")
	return compose.Up(output)
}

func runDown(cmd *cobra.Command, args []string) error {
	volumes, _ := cmd.Flags().GetBool("volumes")

	// Find the compose file
	composeFile := ""
	for _, name := range []string{"docker-compose.dbctl.yml", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if _, err := os.Stat(name); err == nil {
			composeFile = name
			break
		}
	}

	if composeFile == "" {
		return fmt.Errorf("no Docker Compose file found in current directory")
	}

	ui.Info("Stopping containers", "compose", composeFile)
	return compose.Down(composeFile, volumes)
}
