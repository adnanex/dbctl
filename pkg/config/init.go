package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/global.yaml
var GlobalTemplate string

//go:embed templates/local.yaml
var LocalTemplate string

//go:embed templates/minimal.yaml
var MinimalTemplate string

// ResolveInitPath resolves a destination path based on keywords or explicit path.
func ResolveInitPath(dest string) (resolvedPath string, isGlobal bool, err error) {
	home, _ := os.UserHomeDir()

	trimmed := strings.TrimSpace(dest)
	switch strings.ToLower(trimmed) {
	case "", "local", "project", "./", ".":
		return "config.yaml", false, nil
	case "minimal":
		return "config.yaml", false, nil
	case "global", "~", "~/", "~/.dbctl":
		if home == "" {
			return "", true, fmt.Errorf("unable to determine user home directory")
		}
		return filepath.Join(home, ".dbctl", "config.yaml"), true, nil
	case "config", "~/.config", "config-dir", "~/.config/dbctl":
		if home == "" {
			return "", true, fmt.Errorf("unable to determine user home directory")
		}
		return filepath.Join(home, ".config", "dbctl", "config.yaml"), true, nil
	default:
		// Handle path expansion if starts with ~/
		if strings.HasPrefix(trimmed, "~/") {
			if home != "" {
				trimmed = filepath.Join(home, trimmed[2:])
			}
		}
		isGlobal = strings.Contains(trimmed, ".dbctl") || strings.Contains(trimmed, ".config/dbctl")
		return trimmed, isGlobal, nil
	}
}

// GenerateConfigFile writes a template to targetPath, creating parent directories as needed.
func GenerateConfigFile(targetPath string, templateType string, force bool) error {
	if !force {
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("file %s already exists (use --force or -f to overwrite)", targetPath)
		}
	}

	dir := filepath.Dir(targetPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	var content string
	switch strings.ToLower(templateType) {
	case "global":
		content = GlobalTemplate
	case "minimal":
		content = MinimalTemplate
	default:
		content = LocalTemplate
	}

	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", targetPath, err)
	}

	return nil
}
