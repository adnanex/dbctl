package cmd

import (
	_ "embed"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/container"
	"github.com/adnanex/dbctl/pkg/driver"
	"github.com/adnanex/dbctl/pkg/ui"
)

//go:embed longdesc/status.txt
var statusLong string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of configured database targets",
	Long:  strings.TrimRight(statusLong, "\n"),
	Example: `  dbctl status                 # Table view of all targets
  dbctl status -c config.yaml  # Check specific config`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	configFile := cfgFile
	if configFile == "" {
		configFile = "config.yaml"
	}

	cfg, err := config.LoadConfig(configFile, nil)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Load global config
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

	fmt.Println()
	fmt.Println(ui.HeaderStyle.Render("dbctl status"))
	fmt.Println()

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSubtle)
	fmt.Printf("  %s  %s  %s  %s  %s\n",
		headerStyle.Render(fmt.Sprintf("%-10s", "DRIVER")),
		headerStyle.Render(fmt.Sprintf("%-20s", "ENDPOINT")),
		headerStyle.Render(fmt.Sprintf("%-12s", "CONTAINER")),
		headerStyle.Render(fmt.Sprintf("%-5s", "DBS")),
		headerStyle.Render(fmt.Sprintf("%-5s", "USERS")),
	)
	fmt.Println()

	for _, target := range cfg.Targets {
		endpoint := fmt.Sprintf("%s:%d", target.Host, target.Port)

		// Check container status
		containerStatus := ui.Faint.Render("none")
		if target.Container != nil {
			containerStatus = checkContainerStatus(target.Container, target.Driver)
		}

		// Check if driver is available
		_, driverErr := driver.Get(target.Driver)
		driverLabel := ui.DriverBadge(target.Driver)
		if driverErr != nil {
			driverLabel = ui.WarningStyle.Render(target.Driver)
		}

		fmt.Printf("  %s  %s  %s  %s  %s\n",
			fmt.Sprintf("%-10s", driverLabel),
			fmt.Sprintf("%-20s", endpoint),
			fmt.Sprintf("%-12s", containerStatus),
			fmt.Sprintf("%-5d", len(target.Databases)),
			fmt.Sprintf("%-5d", len(target.Users)),
		)
	}

	fmt.Println()
	return nil
}

func checkContainerStatus(cfg *config.ContainerConfig, driverName string) string {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return ui.ErrorStyle.Render("no docker")
	}

	// Check Compose service
	if cfg.Service != "" || cfg.ComposeFile != "" {
		composeFile := container.ResolveComposeFile(cfg.ComposeFile)
		service := cfg.Service
		if service == "" {
			service = driverName
		}

		args := []string{"compose"}
		if composeFile != "" {
			args = append(args, "-f", composeFile)
		}
		args = append(args, "ps", "-q", "--status", "running", service)

		cmd := exec.Command(dockerPath, args...)
		out, _ := cmd.Output()
		if strings.TrimSpace(string(out)) != "" {
			return ui.SuccessStyle.Render("running")
		}
		return ui.WarningStyle.Render("stopped")
	}

	// Check standalone container
	if cfg.Name != "" {
		cmd := exec.Command(dockerPath, "inspect", cfg.Name, "--format", "{{.State.Status}}")
		out, err := cmd.Output()
		if err != nil {
			return ui.ErrorStyle.Render("not found")
		}
		status := fmt.Sprintf("%s", out)
		if status == "running" {
			return ui.SuccessStyle.Render("running")
		}
		return ui.WarningStyle.Render(status)
	}

	return ui.Faint.Render("none")
}
