package driver

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/ui"
)

// Driver defines the interface that all database engine drivers must implement.
type Driver interface {
	Name() string
	Connect(ctx context.Context, target *config.TargetConfig) error
	Close() error
	EnsureDatabase(ctx context.Context, db config.DatabaseConfig) error
	EnsureUser(ctx context.Context, user config.UserConfig) error
	EnsureGrant(ctx context.Context, user config.UserConfig, grant config.GrantConfig) error
}

type Factory func() Driver

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register registers a driver factory under the given name.
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Get returns an instance of the requested driver.
func Get(name string) (Driver, error) {
	registryMu.RLock()
	factory, exists := registry[name]
	registryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported driver %q (available drivers: %v)", name, SupportedDrivers())
	}
	return factory(), nil
}

// SupportedDrivers returns a list of registered driver names.
func SupportedDrivers() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	var names []string
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ProvisionTarget provisions all databases, users, and grants for a target.
func ProvisionTarget(ctx context.Context, target *config.TargetConfig, dryRun bool) error {
	drv, err := Get(target.Driver)
	if err != nil {
		return err
	}

	if dryRun {
		ui.Info("DRY-RUN: simulating changes (no modifications will be made)",
			"driver", target.Driver,
			"host", target.Host,
			"port", target.Port,
		)
	} else {
		ui.Info("Connecting to database",
			"driver", target.Driver,
			"host", target.Host,
			"port", target.Port,
			"admin", target.Admin.Username,
		)
		if err := drv.Connect(ctx, target); err != nil {
			return fmt.Errorf("[%s] connection failed: %w", target.Driver, err)
		}
		defer drv.Close()
	}

	// 1. Ensure databases
	for _, db := range target.Databases {
		if dryRun {
			ui.Info("Would ensure database", "driver", target.Driver, "database", db.Name)
			continue
		}
		if err := drv.EnsureDatabase(ctx, db); err != nil {
			return fmt.Errorf("[%s] error provisioning database %s: %w", target.Driver, db.Name, err)
		}
	}

	// 2. Ensure users & grants
	for _, user := range target.Users {
		if dryRun {
			ui.Info("Would ensure user", "driver", target.Driver, "user", user.Username, "host", user.Host)
			for _, grant := range user.Grants {
				ui.Info("Would grant privileges",
					"driver", target.Driver,
					"privileges", grant.Privileges,
					"database", grant.Database,
					"user", user.Username,
				)
			}
			continue
		}

		if err := drv.EnsureUser(ctx, user); err != nil {
			return fmt.Errorf("[%s] error provisioning user %s: %w", target.Driver, user.Username, err)
		}

		for _, grant := range user.Grants {
			if err := drv.EnsureGrant(ctx, user, grant); err != nil {
				return fmt.Errorf("[%s] error granting %s on %s to %s: %w", target.Driver, grant.Privileges, grant.Database, user.Username, err)
			}
		}
	}

	ui.Info("Successfully provisioned all resources",
		"driver", target.Driver,
		"host", target.Host,
		"port", target.Port,
	)
	return nil
}
