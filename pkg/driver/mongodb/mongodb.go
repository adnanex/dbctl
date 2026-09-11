package mongodb

import (
	"context"
	"fmt"

	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/driver"
)

func init() {
	driver.Register("mongodb", func() driver.Driver {
		return &MongoDriver{}
	})
	driver.Register("mongo", func() driver.Driver {
		return &MongoDriver{}
	})
}

// MongoDriver provides driver implementation scaffolding for MongoDB.
type MongoDriver struct{}

func (m *MongoDriver) Name() string {
	return "mongodb"
}

func (m *MongoDriver) Connect(ctx context.Context, target *config.TargetConfig) error {
	return fmt.Errorf("mongodb driver is scaffolded and will be enabled when mongodb extension is plugged in (started with mysql)")
}

func (m *MongoDriver) Close() error {
	return nil
}

func (m *MongoDriver) EnsureDatabase(ctx context.Context, dbSpec config.DatabaseConfig) error {
	return fmt.Errorf("mongodb driver not yet implemented")
}

func (m *MongoDriver) EnsureUser(ctx context.Context, userSpec config.UserConfig) error {
	return fmt.Errorf("mongodb driver not yet implemented")
}

func (m *MongoDriver) EnsureGrant(ctx context.Context, userSpec config.UserConfig, grant config.GrantConfig) error {
	return fmt.Errorf("mongodb driver not yet implemented")
}
