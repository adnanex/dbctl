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

// projectConfigName is the default filename for a project's own driver config
// ("targets:" or a single root "driver:"). It intentionally isn't the generic
// "config.yaml" — that name collides with unrelated config.yaml files many
// other tools already use in a project, silently mixing up or overwriting them.
const projectConfigName = "dbctl.yaml"

// ResolveInitPath resolves a destination path based on keywords or explicit path.
//
// "local"/"minimal" (and no destination) scaffold a project driver config
// ("targets:" format) at ./dbctl.yaml. "project" (and ".dbctl") scaffold a
// project-local "defaults:" config at ./.dbctl/config.yaml instead — the
// project-scoped equivalent of "global", useful for committing shared,
// non-secret defaults (service names, compose_file wiring) to the repo. See
// FindGlobalConfigFile, which checks ./.dbctl/config.yaml before ~/.dbctl/.
func ResolveInitPath(dest string) (resolvedPath string, isGlobal bool, err error) {
	home, _ := os.UserHomeDir()

	trimmed := strings.TrimSpace(dest)
	switch strings.ToLower(trimmed) {
	case "", "local", "./", ".":
		return projectConfigName, false, nil
	case "minimal":
		return projectConfigName, false, nil
	case "project", ".dbctl", "./.dbctl":
		return filepath.Join(".dbctl", "config.yaml"), true, nil
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

// FindProjectConfigFile resolves the project driver config path, checking (in
// order): an explicit path, ./dbctl.yaml, ./dbctl.yml, ./.dbctl/config.yaml,
// ./.dbctl/config.yml. The .dbctl/ fallback lets a "targets:" config live
// there too, alongside (or instead of) a "defaults:" block — LoadConfig only
// reads the "targets" key, so both can coexist in one file. If none exist, it
// returns the default name so callers can produce a sensible "not found" error.
func FindProjectConfigFile(explicitPath string) string {
	if explicitPath != "" {
		return explicitPath
	}

	candidates := []string{
		projectConfigName,
		"dbctl.yml",
		filepath.Join(".dbctl", "config.yaml"),
		filepath.Join(".dbctl", "config.yml"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return projectConfigName
}

// ResolveStackPath resolves a destination directory for a database stack (docker-compose.yml)
// and the matching config.yaml path, based on keywords or an explicit path.
//
// The stack's companion config.yaml is always "defaults:" format (see
// GenerateStackConfig), so unlike ResolveInitPath, "local" isn't the bare
// project driver config's path — it resolves to ./.dbctl/, same as "project",
// to avoid colliding with ./dbctl.yaml's incompatible "targets:" format.
func ResolveStackPath(dest string) (stackDir string, configPath string, err error) {
	home, _ := os.UserHomeDir()

	trimmed := strings.TrimSpace(dest)
	switch strings.ToLower(trimmed) {
	case "", "local", "./", ".", "project", ".dbctl", "./.dbctl":
		return filepath.Join(".dbctl", "dbs"), filepath.Join(".dbctl", "config.yaml"), nil
	case "global", "~", "~/", "~/.dbctl":
		if home == "" {
			return "", "", fmt.Errorf("unable to determine user home directory")
		}
		return filepath.Join(home, ".dbctl", "dbs"), filepath.Join(home, ".dbctl", "config.yaml"), nil
	case "config", "~/.config", "config-dir", "~/.config/dbctl":
		if home == "" {
			return "", "", fmt.Errorf("unable to determine user home directory")
		}
		return filepath.Join(home, ".config", "dbctl", "dbs"), filepath.Join(home, ".config", "dbctl", "config.yaml"), nil
	default:
		if strings.HasPrefix(trimmed, "~/") {
			if home != "" {
				trimmed = filepath.Join(home, trimmed[2:])
			}
		}
		// Treat an explicit destination as a base directory: the stack lives
		// in its "dbs" subdirectory, with the matching config.yaml alongside it —
		// mirroring the "global"/"config" keyword layout above.
		return filepath.Join(trimmed, "dbs"), filepath.Join(trimmed, "config.yaml"), nil
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
