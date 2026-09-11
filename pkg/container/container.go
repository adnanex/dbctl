package container

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/adnanex/dbctl/pkg/config"
)

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
	composeFile := cfg.ComposeFile
	if composeFile == "" {
		for _, f := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
			if _, err := os.Stat(f); err == nil {
				composeFile = f
				break
			}
		}
	}

	service := cfg.Service
	if service == "" {
		service = driverName
	}

	var args []string
	if composeFile != "" {
		args = append(args, "compose", "-f", composeFile)
	} else {
		args = append(args, "compose")
	}

	// Check status
	checkArgs := append(args, "ps", "--status", "running", service)
	cmd := exec.CommandContext(ctx, dockerPath, checkArgs...)
	out, _ := cmd.CombinedOutput()

	if strings.Contains(string(out), service) {
		log.Printf("[container] Docker Compose service %q is already running.", service)
		return nil
	}

	log.Printf("[container] Starting Docker Compose service %q (file: %s)...", service, composeFile)
	upArgs := append(args, "up", "-d", service)
	upCmd := exec.CommandContext(ctx, dockerPath, upArgs...)
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr
	if err := upCmd.Run(); err != nil {
		return fmt.Errorf("failed to start compose service %q: %w", service, err)
	}

	log.Printf("[container] Started Docker Compose service %q", service)
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
			log.Printf("[container] Container %q is already running.", name)
			return nil
		}
		log.Printf("[container] Container %q exists with status %q. Starting...", name, status)
		startCmd := exec.CommandContext(ctx, dockerPath, "start", name)
		startCmd.Stdout = os.Stdout
		startCmd.Stderr = os.Stderr
		return startCmd.Run()
	}

	// Container doesn't exist, create and run
	log.Printf("[container] Creating and starting container %q from image %q...", name, cfg.Image)
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

	log.Printf("[container] Successfully created and started container %q", name)
	return nil
}

func waitForPort(ctx context.Context, host string, port int, timeout time.Duration) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[container] Waiting for %s to become ready (timeout: %s)...", addr, timeout)

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
			log.Printf("[container] %s is reachable and ready!", addr)
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}
