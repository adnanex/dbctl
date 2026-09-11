package postgres

import (
	"context"
	"fmt"

	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/driver"
)

func init() {
	driver.Register("postgres", func() driver.Driver {
		return &PostgresDriver{}
	})
	driver.Register("postgresql", func() driver.Driver {
		return &PostgresDriver{}
	})
}

// PostgresDriver provides driver implementation scaffolding for PostgreSQL.
type PostgresDriver struct{}

func (p *PostgresDriver) Name() string {
	return "postgres"
}

func (p *PostgresDriver) Connect(ctx context.Context, target *config.TargetConfig) error {
	return fmt.Errorf("postgres driver is scaffolded and will be enabled when postgres extension is plugged in (started with mysql)")
}

func (p *PostgresDriver) Close() error {
	return nil
}

func (p *PostgresDriver) EnsureDatabase(ctx context.Context, dbSpec config.DatabaseConfig) error {
	return fmt.Errorf("postgres driver not yet implemented")
}

func (p *PostgresDriver) EnsureUser(ctx context.Context, userSpec config.UserConfig) error {
	return fmt.Errorf("postgres driver not yet implemented")
}

func (p *PostgresDriver) EnsureGrant(ctx context.Context, userSpec config.UserConfig, grant config.GrantConfig) error {
	return fmt.Errorf("postgres driver not yet implemented")
}
