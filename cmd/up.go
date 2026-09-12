package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adnanex/dbctl/pkg/compose"
	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/container"
	"github.com/adnanex/dbctl/pkg/ui"
)

//go:embed longdesc/up.txt
var upLong string

//go:embed longdesc/down.txt
var downLong string

var upCmd = &cobra.Command{
	Use:   "up [driver...]",
	Short: "Generate Docker Compose and start database containers",
	Long:  strings.TrimRight(upLong, "\n"),
	Example: `  dbctl up                          # Start every target in your config
  dbctl up mysql                    # Start only the mysql target
  dbctl up mysql mongodb            # Start only these targets
  dbctl up --generate-only          # Just generate docker-compose.dbctl.yml
  dbctl up --compose ./my-compose.yml  # Use custom compose file
  dbctl up --detach                 # Run in background (default)`,
	Args: cobra.ArbitraryArgs,
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

	driverFilter := normalizeDriverFilter(args)

	// If no explicit --compose flag was given, check whether a host/project stack
	// (DBCTL_COMPOSE_FILE, DBCTL_STACK, ~/.dbctl/dbs, ./dbs, ...) already exists
	// before falling back to generating a fresh compose file from config.
	if customCompose == "" {
		if discovered := container.ResolveComposeFile(""); discovered != "" {
			if _, err := os.Stat(discovered); err == nil {
				customCompose = discovered
			}
		}
	}

	// If using a custom (or discovered) compose file, only start the services
	// this project's own config actually needs — a discovered file is often a
	// shared, multi-engine stack, and this project may only use one of them —
	// narrowed further by an explicit driver filter, if given.
	if customCompose != "" {
		if _, err := os.Stat(customCompose); os.IsNotExist(err) {
			return fmt.Errorf("compose file not found: %s", customCompose)
		}

		services, err := resolveComposeServices(driverFilter)
		if err != nil {
			return err
		}

		ui.Info("Using compose file", "path", customCompose)
		if !generateOnly {
			return compose.Up(customCompose, services...)
		}
		return nil
	}

	// Load config to generate compose
	configFile := config.FindProjectConfigFile(cfgFile)

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

	if len(driverFilter) > 0 {
		filtered, err := filterTargetsByDriver(cfg.Targets, driverFilter)
		if err != nil {
			return fmt.Errorf("%w (in %s)", err, configFile)
		}
		cfg.Targets = filtered
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

// normalizeDriverFilter canonicalizes CLI driver-name arguments (lowercased,
// with the same postgres/postgresql and mongo/mongodb aliases LoadConfig
// applies to targets) so they compare equal to already-normalized drivers.
func normalizeDriverFilter(args []string) []string {
	out := make([]string, 0, len(args))
	seen := make(map[string]bool, len(args))
	for _, a := range args {
		d := strings.ToLower(strings.TrimSpace(a))
		switch d {
		case "postgresql":
			d = "postgres"
		case "mongo":
			d = "mongodb"
		}
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

// filterTargetsByDriver returns only the targets matching driverFilter,
// erroring if any requested driver matches nothing.
func filterTargetsByDriver(targets []config.TargetConfig, driverFilter []string) ([]config.TargetConfig, error) {
	want := make(map[string]bool, len(driverFilter))
	for _, d := range driverFilter {
		want[d] = true
	}

	matched := make(map[string]bool, len(driverFilter))
	filtered := make([]config.TargetConfig, 0, len(targets))
	for _, t := range targets {
		if want[t.Driver] {
			matched[t.Driver] = true
			filtered = append(filtered, t)
		}
	}

	for _, d := range driverFilter {
		if !matched[d] {
			return nil, fmt.Errorf("no target with driver %q", d)
		}
	}

	return filtered, nil
}

// resolveComposeServices determines which Compose services `up` should start
// when using a custom or auto-discovered compose file, which may be a shared,
// multi-engine stack unrelated to what the current project actually needs. It
// loads the project's own config (if any) and combines:
//   - each target's driver, mapped to its wired container service
//     (config.ContainerConfig.Service, defaulting to the driver name)
//   - each entry in config.Config.Services, a plain Compose service name for
//     a container-only engine (redis, kafka, ...) that isn't a target
//
// so `dbctl up` only starts what this project declares — narrowed further to
// selector when non-empty (matched against target drivers and Services
// entries alike).
//
// If no project config can be loaded, selector (if any) is returned as-is so
// its values are used as literal Compose service names; an empty result
// tells compose.Up to start every service, preserving prior behavior for a
// bare `--compose` pointing at a file with no associated dbctl config.
func resolveComposeServices(selector []string) ([]string, error) {
	configFile := config.FindProjectConfigFile(cfgFile)
	cfg, err := config.LoadConfig(configFile, nil)
	if err != nil {
		return selector, nil
	}

	if len(cfg.Targets) == 0 && len(cfg.Services) == 0 {
		return nil, fmt.Errorf("no target databases or services defined in %s", configFile)
	}

	globalPath := config.FindGlobalConfigFile("")
	if globalPath != "" {
		if globalCfg, err := config.LoadGlobalConfig(globalPath, nil); err == nil {
			config.MergeGlobalDefaults(cfg, globalCfg)
		}
	}

	// name is what a selector arg matches against; service is what's passed
	// to `docker compose up -d`.
	type wanted struct{ name, service string }
	all := make([]wanted, 0, len(cfg.Targets)+len(cfg.Services))
	for _, t := range cfg.Targets {
		service := t.Driver
		if t.Container != nil && t.Container.Service != "" {
			service = t.Container.Service
		}
		all = append(all, wanted{name: t.Driver, service: service})
	}
	for _, s := range cfg.Services {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		all = append(all, wanted{name: strings.ToLower(s), service: s})
	}

	if len(selector) > 0 {
		want := make(map[string]bool, len(selector))
		for _, d := range selector {
			want[d] = true
		}
		matched := make(map[string]bool, len(selector))
		filtered := make([]wanted, 0, len(all))
		for _, w := range all {
			if want[w.name] {
				matched[w.name] = true
				filtered = append(filtered, w)
			}
		}
		for _, d := range selector {
			if !matched[d] {
				return nil, fmt.Errorf("no target or service %q in %s", d, configFile)
			}
		}
		all = filtered
	}

	seen := make(map[string]bool, len(all))
	services := make([]string, 0, len(all))
	for _, w := range all {
		if !seen[w.service] {
			seen[w.service] = true
			services = append(services, w.service)
		}
	}

	return services, nil
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
