package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/container"
	"github.com/adnanex/dbctl/pkg/driver"
	"github.com/adnanex/dbctl/pkg/ui"

	_ "github.com/adnanex/dbctl/pkg/driver/mongodb"
	_ "github.com/adnanex/dbctl/pkg/driver/mysql"
	_ "github.com/adnanex/dbctl/pkg/driver/postgres"
)

//go:embed longdesc/provision.txt
var provisionLong string

var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision databases, users, and permissions from config",
	Long:  strings.TrimRight(provisionLong, "\n"),
	Example: `  dbctl provision                          # Use ./dbctl.yaml
  dbctl provision -c my-config.yaml        # Specify config file
  dbctl provision --dry-run                # Simulate without changes
  dbctl provision -e .env.mysql -e .env    # Load env files
  dbctl provision --skip-container         # Don't auto-start containers`,
	Aliases: []string{"apply", "run"},
	RunE:    runProvision,
}

func init() {
	rootCmd.AddCommand(provisionCmd)

	// Make `dbctl` (no subcommand) behave like `dbctl provision` for backward compatibility
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// If no subcommand is specified and no args, run provision
		if len(args) == 0 {
			return runProvision(cmd, args)
		}
		return cmd.Help()
	}
}

func runProvision(cmd *cobra.Command, args []string) error {
	dryRun := viper.GetBool("dry-run")
	skipContainer := viper.GetBool("skip-container")
	timeout := viper.GetDuration("timeout")
	envFiles := viper.GetStringSlice("env")
	globalConfigPath := viper.GetString("global-config")

	// Resolve config file path
	configFile := config.FindProjectConfigFile(viper.GetString("config"))

	// Auto-detect .env file
	if len(envFiles) == 0 {
		if _, err := os.Stat(".env"); err == nil {
			ui.Info("Detected local .env file, loading automatically")
			envFiles = append(envFiles, ".env")
		}
	}

	// Load env files
	var envMap map[string]string
	if len(envFiles) > 0 {
		ui.Info("Loading environment variables", "files", envFiles)
		var err error
		envMap, err = config.LoadEnvFiles(envFiles)
		if err != nil {
			return fmt.Errorf("error loading env file(s): %w", err)
		}
		ui.Info("Loaded environment variables", "count", len(envMap))
	}

	// Discover and load global configuration
	globalPath := config.FindGlobalConfigFile(globalConfigPath)
	var globalCfg *config.GlobalConfig
	if globalPath != "" {
		ui.Info("Loading global configuration", "path", globalPath)
		var err error
		globalCfg, err = config.LoadGlobalConfig(globalPath, envMap)
		if err != nil {
			ui.Warn("Failed to load global config", "path", globalPath, "err", err)
		} else {
			ui.Info("Loaded driver defaults from global config", "count", len(globalCfg.Defaults))
		}
	}

	// Load project configuration
	ui.Info("Loading project configuration", "path", configFile)
	cfg, err := config.LoadConfig(configFile, envMap)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Merge global defaults
	if globalCfg != nil {
		config.MergeGlobalDefaults(cfg, globalCfg)
	}

	if len(cfg.Targets) == 0 {
		return fmt.Errorf("no target databases defined in %s", configFile)
	}

	ui.Info("Found target databases",
		"count", len(cfg.Targets),
		"drivers", driver.SupportedDrivers(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	successCount := 0
	for i, target := range cfg.Targets {
		fmt.Println()
		header := fmt.Sprintf("── [%d/%d] %s on %s:%d ",
			i+1, len(cfg.Targets),
			ui.DriverBadge(target.Driver),
			target.Host, target.Port,
		)
		fmt.Println(ui.TargetHeaderStyle.Render(header))

		// Check and start container if configured
		if !skipContainer && !dryRun && target.Container != nil {
			var containerErr error
			err := ui.RunWithSpinnerf("Starting %s container...", func() error {
				containerErr = container.EnsureContainerRunning(ctx, target.Container, target.Driver, target.Host, target.Port)
				return containerErr
			}, target.Driver)
			if err != nil {
				return fmt.Errorf("container startup failed for %s: %w", target.Driver, err)
			}
		}

		if err := driver.ProvisionTarget(ctx, &target, dryRun); err != nil {
			ui.Error("Provisioning failed",
				"driver", target.Driver,
				"host", target.Host,
				"port", target.Port,
				"err", err,
			)
			os.Exit(1)
		}
		successCount++
	}

	fmt.Println()
	if dryRun {
		ui.Info(fmt.Sprintf("%s Dry run complete — %d/%d target(s) simulated",
			ui.SuccessStyle.Render("✓"),
			successCount, len(cfg.Targets),
		))
	} else {
		ui.Info(fmt.Sprintf("%s Successfully provisioned %d/%d database target(s)",
			ui.SuccessStyle.Render("✓"),
			successCount, len(cfg.Targets),
		))
	}

	// Suppress unused import warnings for time
	_ = time.Second

	return nil
}
