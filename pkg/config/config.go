package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level configuration file.
// It supports either multiple targets under `targets`, or a single target defined at the root.
type Config struct {
	Targets []TargetConfig `yaml:"targets"`
}

// TargetConfig defines connection settings and resources for a specific database engine.
type TargetConfig struct {
	Driver        string           `yaml:"driver"`
	Host          string           `yaml:"host"`
	Port          int              `yaml:"port"`
	Admin         AdminConfig      `yaml:"admin"`
	AdminUser     string           `yaml:"admin_user"`     // Flat alias for admin.username
	AdminPassword string           `yaml:"admin_password"` // Flat alias for admin.password
	Databases     []DatabaseConfig `yaml:"databases"`
	Users         []UserConfig     `yaml:"users"`
	Container     *ContainerConfig `yaml:"container,omitempty"`
}

// ContainerConfig specifies container orchestration options (Compose or standalone Docker).
type ContainerConfig struct {
	ComposeFile string            `yaml:"compose_file"`
	Service     string            `yaml:"service"`
	Image       string            `yaml:"image"`
	Name        string            `yaml:"name"`
	Ports       []string          `yaml:"ports"`
	Env         map[string]string `yaml:"env"`
	Volumes     []string          `yaml:"volumes"`
	AutoRemove  bool              `yaml:"auto_remove"`
	Command     string            `yaml:"command"`
	WaitTimeout string            `yaml:"wait_timeout"` // e.g. "60s"
}

// GlobalConfig represents the system-wide or user-level default configuration.
type GlobalConfig struct {
	Defaults map[string]TargetConfig `yaml:"defaults"`
	Targets  []TargetConfig          `yaml:"targets,omitempty"`
}

// AdminConfig holds the admin credentials used to provision databases and users.
type AdminConfig struct {
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Database   string `yaml:"database"`    // Admin db (e.g. "postgres" or "admin")
	AuthSource string `yaml:"auth_source"` // For mongodb (e.g. "admin")
}

// DatabaseConfig specifies a database to ensure exists.
type DatabaseConfig struct {
	Name      string `yaml:"name"`
	Charset   string `yaml:"charset,omitempty"`
	Collation string `yaml:"collation,omitempty"`
}

// UnmarshalYAML allows DatabaseConfig to be unmarshaled from either a string ("mydb") or an object.
func (d *DatabaseConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		d.Name = value.Value
		return nil
	}

	type rawDatabaseConfig DatabaseConfig
	var raw rawDatabaseConfig
	if err := value.Decode(&raw); err != nil {
		return err
	}
	*d = DatabaseConfig(raw)
	return nil
}

// UserConfig specifies a user and their database access permissions.
type UserConfig struct {
	Username  string        `yaml:"username"`
	Password  string        `yaml:"password"`
	Host      string        `yaml:"host"` // Applicable for MySQL (default: "%")
	Databases []string      `yaml:"databases,omitempty"` // Shorthand: databases user gets ALL access to
	Grants    []GrantConfig `yaml:"grants,omitempty"`
}

// GrantConfig defines permissions a user has on a specific database.
type GrantConfig struct {
	Database   string     `yaml:"database"`
	Privileges Privileges `yaml:"privileges"`
}

// Privileges supports either a single string ("ALL") or a list of strings (["SELECT", "INSERT"]).
type Privileges []string

func (p *Privileges) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*p = []string{value.Value}
		return nil
	}
	var list []string
	if err := value.Decode(&list); err != nil {
		return err
	}
	*p = list
	return nil
}

// envRegex matches ${VAR} or ${VAR:-default}
var envRegex = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-([^}]*))?\}`)

// ParseEnvFile reads an env file and returns key-value pairs.
func ParseEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read env file %s: %w", path, err)
	}

	envMap := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Support optional "export KEY=VAL"
		if strings.HasPrefix(trimmed, "export ") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue // skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Handle quoted values
		if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") && len(val) >= 2) ||
			(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") && len(val) >= 2) {
			val = val[1 : len(val)-1]
		} else {
			// Strip inline comments for unquoted values
			if idx := strings.Index(val, " #"); idx != -1 {
				val = strings.TrimSpace(val[:idx])
			}
		}

		if key != "" {
			envMap[key] = val
		}
		_ = lineNum
	}

	return envMap, nil
}

// LoadEnvFiles loads multiple env files in order. Later files override earlier ones.
func LoadEnvFiles(paths []string) (map[string]string, error) {
	combined := make(map[string]string)
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		m, err := ParseEnvFile(p)
		if err != nil {
			return nil, err
		}
		for k, v := range m {
			combined[k] = v
		}
	}
	return combined, nil
}

// expandEnvVars expands environment variables in the format ${VAR} or ${VAR:-default}
// using the provided envMap first, falling back to os.LookupEnv, and then default values.
func expandEnvVars(content []byte, envMap map[string]string) []byte {
	return envRegex.ReplaceAllFunc(content, func(b []byte) []byte {
		matches := envRegex.FindSubmatch(b)
		if len(matches) < 2 {
			return b
		}
		varName := string(matches[1])

		// 1. Check custom envMap from env files
		if envMap != nil {
			if val, exists := envMap[varName]; exists && val != "" {
				return []byte(val)
			}
		}

		// 2. Check system environment
		val, exists := os.LookupEnv(varName)
		if exists && val != "" {
			return []byte(val)
		}

		// 3. Fallback to default in ${VAR:-default}
		if len(matches) >= 3 && matches[2] != nil {
			return matches[2]
		}

		if exists {
			return []byte(val)
		}
		return []byte("")
	})
}

// LoadConfig reads and parses the YAML configuration file from the given path.
func LoadConfig(path string, envMap map[string]string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	data = expandEnvVars(data, envMap)

	// First try parsing as a multi-target configuration
	var root Config
	if err := yaml.Unmarshal(data, &root); err == nil && len(root.Targets) > 0 {
		for i := range root.Targets {
			normalizeTarget(&root.Targets[i])
		}
		return &root, nil
	}

	// Otherwise, attempt parsing as a single-target configuration at the root
	var single TargetConfig
	if err := yaml.Unmarshal(data, &single); err == nil && single.Driver != "" {
		normalizeTarget(&single)
		return &Config{Targets: []TargetConfig{single}}, nil
	}

	return nil, fmt.Errorf("invalid config format in %s: must specify 'driver' or 'targets'", path)
}

// FindGlobalConfigFile searches for the global configuration file.
func FindGlobalConfigFile(explicitPath string) string {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err == nil {
			return explicitPath
		}
		return explicitPath
	}

	if envPath := os.Getenv("DBCTL_GLOBAL_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		candidates := []string{
			filepath.Join(home, ".dbctl", "config.yaml"),
			filepath.Join(home, ".dbctl", "config.yml"),
			filepath.Join(home, ".config", "dbctl", "config.yaml"),
			filepath.Join(home, ".config", "dbctl", "config.yml"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	return ""
}

// LoadGlobalConfig loads global configuration and returns the parsed GlobalConfig.
func LoadGlobalConfig(path string, envMap map[string]string) (*GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config %s: %w", path, err)
	}

	data = expandEnvVars(data, envMap)

	var g GlobalConfig
	if err := yaml.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("failed to parse global config %s: %w", path, err)
	}

	if g.Defaults == nil {
		g.Defaults = make(map[string]TargetConfig)
	}

	// Also support flat root-level drivers in global config (e.g. mysql: {...})
	var rawRoot map[string]yaml.Node
	if err := yaml.Unmarshal(data, &rawRoot); err == nil {
		for key, node := range rawRoot {
			keyLower := strings.ToLower(strings.TrimSpace(key))
			if keyLower == "defaults" || keyLower == "targets" {
				continue
			}
			if keyLower == "mysql" || keyLower == "postgres" || keyLower == "postgresql" || keyLower == "mongo" || keyLower == "mongodb" {
				var target TargetConfig
				if err := node.Decode(&target); err == nil {
					target.Driver = keyLower
					normalizeTarget(&target)
					g.Defaults[keyLower] = target
				}
			}
		}
	}

	for k, v := range g.Defaults {
		v.Driver = strings.ToLower(strings.TrimSpace(k))
		normalizeTarget(&v)
		g.Defaults[strings.ToLower(strings.TrimSpace(k))] = v
	}

	return &g, nil
}

// MergeGlobalDefaults merges global defaults into targets where values are omitted.
func MergeGlobalDefaults(cfg *Config, global *GlobalConfig) {
	if global == nil || len(global.Defaults) == 0 {
		return
	}

	for i := range cfg.Targets {
		t := &cfg.Targets[i]
		driverKey := strings.ToLower(strings.TrimSpace(t.Driver))
		def, exists := global.Defaults[driverKey]
		if !exists {
			if driverKey == "postgresql" {
				def, exists = global.Defaults["postgres"]
			} else if driverKey == "postgres" {
				def, exists = global.Defaults["postgresql"]
			} else if driverKey == "mongo" {
				def, exists = global.Defaults["mongodb"]
			} else if driverKey == "mongodb" {
				def, exists = global.Defaults["mongo"]
			}
		}

		if exists {
			if t.Host == "" {
				t.Host = def.Host
			}
			if t.Port == 0 {
				t.Port = def.Port
			}
			if t.Admin.Username == "" {
				t.Admin.Username = def.Admin.Username
			}
			if t.Admin.Password == "" {
				t.Admin.Password = def.Admin.Password
			}
			if t.Admin.Database == "" {
				t.Admin.Database = def.Admin.Database
			}
			if t.Admin.AuthSource == "" {
				t.Admin.AuthSource = def.Admin.AuthSource
			}

			// Merge container config
			if def.Container != nil {
				if t.Container == nil {
					copyContainer := *def.Container
					t.Container = &copyContainer
				} else {
					if t.Container.ComposeFile == "" {
						t.Container.ComposeFile = def.Container.ComposeFile
					}
					if t.Container.Service == "" {
						t.Container.Service = def.Container.Service
					}
					if t.Container.Image == "" {
						t.Container.Image = def.Container.Image
					}
					if t.Container.Name == "" {
						t.Container.Name = def.Container.Name
					}
					if len(t.Container.Ports) == 0 {
						t.Container.Ports = def.Container.Ports
					}
					if len(t.Container.Env) == 0 && len(def.Container.Env) > 0 {
						t.Container.Env = make(map[string]string)
						for k, v := range def.Container.Env {
							t.Container.Env[k] = v
						}
					}
					if t.Container.WaitTimeout == "" {
						t.Container.WaitTimeout = def.Container.WaitTimeout
					}
				}
			}
		}

		normalizeTarget(t)
	}
}

// normalizeTarget sets defaults and handles aliases for TargetConfig.
func normalizeTarget(t *TargetConfig) {
	t.Driver = strings.ToLower(strings.TrimSpace(t.Driver))

	// Handle admin flat aliases
	if t.Admin.Username == "" && t.AdminUser != "" {
		t.Admin.Username = t.AdminUser
	}
	if t.Admin.Password == "" && t.AdminPassword != "" {
		t.Admin.Password = t.AdminPassword
	}

	// Driver specific defaults
	switch t.Driver {
	case "mysql":
		if t.Host == "" {
			t.Host = "127.0.0.1"
		}
		if t.Port == 0 {
			t.Port = 3306
		}
		if t.Admin.Username == "" {
			t.Admin.Username = "root"
		}
		for i := range t.Databases {
			if t.Databases[i].Charset == "" {
				t.Databases[i].Charset = "utf8mb4"
			}
			if t.Databases[i].Collation == "" {
				t.Databases[i].Collation = "utf8mb4_unicode_ci"
			}
		}
		for i := range t.Users {
			if t.Users[i].Host == "" {
				t.Users[i].Host = "%"
			}
			// Convert shorthand databases to grants
			for _, dbName := range t.Users[i].Databases {
				hasGrant := false
				for _, g := range t.Users[i].Grants {
					if g.Database == dbName {
						hasGrant = true
						break
					}
				}
				if !hasGrant {
					t.Users[i].Grants = append(t.Users[i].Grants, GrantConfig{
						Database:   dbName,
						Privileges: []string{"ALL PRIVILEGES"},
					})
				}
			}
			// Normalize grant privileges
			for j := range t.Users[i].Grants {
				if len(t.Users[i].Grants[j].Privileges) == 0 {
					t.Users[i].Grants[j].Privileges = []string{"ALL PRIVILEGES"}
				}
			}
		}
	case "postgres", "postgresql":
		t.Driver = "postgres"
		if t.Host == "" {
			t.Host = "127.0.0.1"
		}
		if t.Port == 0 {
			t.Port = 5432
		}
		if t.Admin.Username == "" {
			t.Admin.Username = "postgres"
		}
		if t.Admin.Database == "" {
			t.Admin.Database = "postgres"
		}
	case "mongo", "mongodb":
		t.Driver = "mongodb"
		if t.Host == "" {
			t.Host = "127.0.0.1"
		}
		if t.Port == 0 {
			t.Port = 27017
		}
		if t.Admin.AuthSource == "" {
			t.Admin.AuthSource = "admin"
		}
	}
}
