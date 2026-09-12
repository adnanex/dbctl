package container

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/ui"
)

// ResolveComposeFile resolves which Docker Compose file to use, checking in order:
//  1. explicitPath, if it points to an existing file (e.g. a target's configured compose_file)
//  2. the host environment variables DBCTL_COMPOSE_FILE (a file) or DBCTL_STACK (a directory)
//  3. project-local ./.dbctl/dbs/docker-compose.yml
//  4. user home locations: ~/.dbctl/dbs/docker-compose.yml, ~/.config/dbctl/dbs/docker-compose.yml
//  5. local locations: ./dbs/docker-compose.yml, ./docker-compose.yml
//
// This lets any project seamlessly discover a host-wide database stack without
// hardcoding its location in project-level configuration.
func ResolveComposeFile(explicitPath string) string {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err == nil {
			return explicitPath
		}
	}

	composeFileNames := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}

	if envFile := os.Getenv("DBCTL_COMPOSE_FILE"); envFile != "" {
		if _, err := os.Stat(envFile); err == nil {
			return envFile
		}
	}

	if stackDir := os.Getenv("DBCTL_STACK"); stackDir != "" {
		for _, name := range composeFileNames {
			candidate := filepath.Join(stackDir, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	for _, name := range composeFileNames {
		candidate := filepath.Join(".dbctl", "dbs", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		for _, dir := range []string{
			filepath.Join(home, ".dbctl", "dbs"),
			filepath.Join(home, ".config", "dbctl", "dbs"),
		} {
			for _, name := range composeFileNames {
				candidate := filepath.Join(dir, name)
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
		}
	}

	for _, candidate := range append(
		[]string{filepath.Join("dbs", "docker-compose.yml"), filepath.Join("dbs", "docker-compose.yaml")},
		composeFileNames...,
	) {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return explicitPath
}

// EnsureContainerRunning starts the container (via Compose or Docker run) if not already active
// and waits until the target port is accepting connections.
func EnsureContainerRunning(ctx context.Context, cfg *config.ContainerConfig, driverName, host string, port int) error {
	if cfg == nil {
		return nil
	}

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker executable not found in PATH: %w", err)
	}

	// 1. Determine mode: Docker Compose vs Standalone Docker
	if cfg.Service != "" || cfg.ComposeFile != "" {
		if err := ensureComposeService(ctx, dockerPath, cfg, driverName); err != nil {
			return err
		}
	} else if cfg.Image != "" {
		if err := ensureStandaloneContainer(ctx, dockerPath, cfg, driverName, port); err != nil {
			return err
		}
	} else {
		return nil
	}

	// 2. Wait for container / port readiness
	waitDuration := 45 * time.Second
	if cfg.WaitTimeout != "" {
		if d, err := time.ParseDuration(cfg.WaitTimeout); err == nil && d > 0 {
			waitDuration = d
		}
	}

	return waitForPort(ctx, host, port, waitDuration)
}

func ensureComposeService(ctx context.Context, dockerPath string, cfg *config.ContainerConfig, driverName string) error {
	composeFile := ResolveComposeFile(cfg.ComposeFile)

	service := cfg.Service
	if service == "" {
		service = driverName
	}

	baseArgs := []string{"compose"}
	if composeFile != "" {
		baseArgs = append(baseArgs, "-f", composeFile)
	}

	// Check status via -q: prints the running container ID, or nothing if not running.
	checkArgs := append(append([]string{}, baseArgs...), "ps", "-q", "--status", "running", service)
	cmd := exec.CommandContext(ctx, dockerPath, checkArgs...)
	out, _ := cmd.Output()

	if strings.TrimSpace(string(out)) != "" {
		ui.Info("Compose service already running", "service", service)
		return nil
	}

	ui.Info("Starting Compose service", "service", service, "file", composeFile)
	upArgs := append(append([]string{}, baseArgs...), "up", "-d", service)
	upCmd := exec.CommandContext(ctx, dockerPath, upArgs...)
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr
	if err := upCmd.Run(); err != nil {
		return fmt.Errorf("failed to start compose service %q: %w", service, err)
	}

	ui.Info("Compose service started", "service", service)
	return nil
}

func ensureStandaloneContainer(ctx context.Context, dockerPath string, cfg *config.ContainerConfig, driverName string, defaultPort int) error {
	name := cfg.Name
	if name == "" {
		name = fmt.Sprintf("dbs-%s", driverName)
	}

	// Check if container exists
	checkCmd := exec.CommandContext(ctx, dockerPath, "inspect", name, "--format", "{{.State.Status}}")
	var out bytes.Buffer
	checkCmd.Stdout = &out
	err := checkCmd.Run()
	status := strings.TrimSpace(out.String())

	if err == nil {
		if status == "running" {
			ui.Info("Container already running", "name", name)
			return nil
		}
		ui.Info("Starting existing container", "name", name, "status", status)
		startCmd := exec.CommandContext(ctx, dockerPath, "start", name)
		startCmd.Stdout = os.Stdout
		startCmd.Stderr = os.Stderr
		return startCmd.Run()
	}

	// Container doesn't exist, create and run
	ui.Info("Creating container", "name", name, "image", cfg.Image)
	runArgs := []string{"run", "-d", "--name", name}

	if cfg.AutoRemove {
		runArgs = append(runArgs, "--rm")
	}

	// Ports
	if len(cfg.Ports) > 0 {
		for _, p := range cfg.Ports {
			runArgs = append(runArgs, "-p", p)
		}
	} else if defaultPort > 0 {
		runArgs = append(runArgs, "-p", fmt.Sprintf("%d:%d", defaultPort, defaultPort))
	}

	// Environment variables
	for k, v := range cfg.Env {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Volumes
	for _, v := range cfg.Volumes {
		runArgs = append(runArgs, "-v", v)
	}

	runArgs = append(runArgs, cfg.Image)
	if cfg.Command != "" {
		runArgs = append(runArgs, strings.Fields(cfg.Command)...)
	}

	runCmd := exec.CommandContext(ctx, dockerPath, runArgs...)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("failed to run container %q: %w", name, err)
	}

	ui.Info("Container created and started", "name", name)
	return nil
}

func waitForPort(ctx context.Context, host string, port int, timeout time.Duration) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	ui.Info("Waiting for port readiness", "addr", addr, "timeout", timeout)

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for %s to become ready", timeout, addr)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := net.DialTimeout("tcp", addr, 1500*time.Millisecond)
		if err == nil {
			conn.Close()
			// Short stabilization pause for DB engine handshake
			time.Sleep(1 * time.Second)
			ui.Info("Port is reachable and ready", "addr", addr)
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}
