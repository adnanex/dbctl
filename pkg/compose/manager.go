package compose

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/adnanex/dbctl/pkg/ui"
)

// Up runs docker compose up -d with the given compose file.
func Up(composeFile string) error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker executable not found in PATH: %w", err)
	}

	ui.Info("Running docker compose up", "file", composeFile)

	cmd := exec.Command(dockerPath, "compose", "-f", composeFile, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up failed: %w", err)
	}

	ui.Info(fmt.Sprintf("%s Containers started", ui.StatusIcon(true)))
	return nil
}

// Down runs docker compose down with the given compose file.
// If volumes is true, it also removes data volumes.
func Down(composeFile string, volumes bool) error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker executable not found in PATH: %w", err)
	}

	args := []string{"compose", "-f", composeFile, "down"}
	if volumes {
		args = append(args, "--volumes")
		ui.Warn("Removing data volumes — this will destroy all database data!")
	}

	cmd := exec.Command(dockerPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose down failed: %w", err)
	}

	ui.Info(fmt.Sprintf("%s Containers stopped", ui.StatusIcon(true)))
	return nil
}

// Ps runs docker compose ps for the given compose file and returns the output.
func Ps(composeFile string) (string, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return "", fmt.Errorf("docker executable not found in PATH: %w", err)
	}

	cmd := exec.Command(dockerPath, "compose", "-f", composeFile, "ps")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker compose ps failed: %w", err)
	}

	return string(out), nil
}
