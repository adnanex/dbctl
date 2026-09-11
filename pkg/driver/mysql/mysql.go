package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/driver"
)

func init() {
	driver.Register("mysql", func() driver.Driver {
		return &MySQLDriver{}
	})
}

// MySQLDriver implements driver.Driver for MySQL.
type MySQLDriver struct {
	db *sql.DB
}

func (m *MySQLDriver) Name() string {
	return "mysql"
}

func (m *MySQLDriver) Connect(ctx context.Context, target *config.TargetConfig) error {
	// DSN format: [username[:password]@][protocol[(address)]]/[?param1=value1&...&paramN=valueN]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?multiStatements=true&parseTime=true&timeout=10s",
		target.Admin.Username,
		target.Admin.Password,
		target.Host,
		target.Port,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// Retry ping a few times in case container was just started and is initializing
	var pingErr error
	for attempt := 1; attempt <= 5; attempt++ {
		pingErr = db.PingContext(ctx)
		if pingErr == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if pingErr != nil {
		db.Close()
		return fmt.Errorf("failed to ping mysql server at %s:%d: %w", target.Host, target.Port, pingErr)
	}

	m.db = db
	return nil
}

func (m *MySQLDriver) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func escapeIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func escapeLiteral(val string) string {
	return "'" + strings.ReplaceAll(val, "'", "''") + "'"
}

// EnsureDatabase creates the database if it doesn't already exist.
func (m *MySQLDriver) EnsureDatabase(ctx context.Context, dbSpec config.DatabaseConfig) error {
	var exists int
	checkQuery := "SELECT 1 FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?"
	err := m.db.QueryRowContext(ctx, checkQuery, dbSpec.Name).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check database %s: %w", dbSpec.Name, err)
	}

	charset := dbSpec.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	collation := dbSpec.Collation
	if collation == "" {
		collation = "utf8mb4_unicode_ci"
	}

	if err == sql.ErrNoRows {
		log.Printf("[mysql] Database %s does not exist. Creating...", dbSpec.Name)
		createQuery := fmt.Sprintf("CREATE DATABASE %s CHARACTER SET %s COLLATE %s",
			escapeIdentifier(dbSpec.Name),
			charset,
			collation,
		)
		if _, err := m.db.ExecContext(ctx, createQuery); err != nil {
			return fmt.Errorf("failed to create database %s: %w", dbSpec.Name, err)
		}
		log.Printf("[mysql] Successfully created database %s", dbSpec.Name)
	} else {
		log.Printf("[mysql] Database %s already exists. Skipping creation.", dbSpec.Name)
	}

	return nil
}

// EnsureUser creates the user if missing, or synchronizes password if already existing.
func (m *MySQLDriver) EnsureUser(ctx context.Context, userSpec config.UserConfig) error {
	host := userSpec.Host
	if host == "" {
		host = "%"
	}

	var exists int
	checkQuery := "SELECT 1 FROM mysql.user WHERE user = ? AND host = ?"
	err := m.db.QueryRowContext(ctx, checkQuery, userSpec.Username, host).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check user %s@%s: %w", userSpec.Username, host, err)
	}

	userLiteral := escapeLiteral(userSpec.Username)
	hostLiteral := escapeLiteral(host)
	passLiteral := escapeLiteral(userSpec.Password)

	if err == sql.ErrNoRows {
		log.Printf("[mysql] User %s@%s does not exist. Creating...", userSpec.Username, host)
		createQuery := fmt.Sprintf("CREATE USER %s@%s IDENTIFIED BY %s",
			userLiteral,
			hostLiteral,
			passLiteral,
		)
		if _, err := m.db.ExecContext(ctx, createQuery); err != nil {
			return fmt.Errorf("failed to create user %s@%s: %w", userSpec.Username, host, err)
		}
		log.Printf("[mysql] Successfully created user %s@%s", userSpec.Username, host)
	} else {
		log.Printf("[mysql] User %s@%s already exists. Synchronizing password...", userSpec.Username, host)
		alterQuery := fmt.Sprintf("ALTER USER %s@%s IDENTIFIED BY %s",
			userLiteral,
			hostLiteral,
			passLiteral,
		)
		if _, err := m.db.ExecContext(ctx, alterQuery); err != nil {
			return fmt.Errorf("failed to alter user %s@%s: %w", userSpec.Username, host, err)
		}
		log.Printf("[mysql] Successfully verified/updated user %s@%s", userSpec.Username, host)
	}

	return nil
}

// EnsureGrant idempotently grants specified privileges on the database to the user.
func (m *MySQLDriver) EnsureGrant(ctx context.Context, userSpec config.UserConfig, grant config.GrantConfig) error {
	host := userSpec.Host
	if host == "" {
		host = "%"
	}

	privs := strings.Join(grant.Privileges, ", ")
	if strings.TrimSpace(privs) == "" {
		privs = "ALL PRIVILEGES"
	}

	var targetResource string
	if grant.Database == "*" {
		targetResource = "*.*"
	} else {
		targetResource = fmt.Sprintf("%s.*", escapeIdentifier(grant.Database))
	}

	grantQuery := fmt.Sprintf("GRANT %s ON %s TO %s@%s",
		privs,
		targetResource,
		escapeLiteral(userSpec.Username),
		escapeLiteral(host),
	)

	log.Printf("[mysql] Granting [%s] on %s to %s@%s...", privs, targetResource, userSpec.Username, host)
	if _, err := m.db.ExecContext(ctx, grantQuery); err != nil {
		return fmt.Errorf("failed to execute grant query (%s): %w", grantQuery, err)
	}

	if _, err := m.db.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		return fmt.Errorf("failed to flush privileges: %w", err)
	}

	log.Printf("[mysql] Successfully granted privileges on %s to %s@%s", targetResource, userSpec.Username, host)
	return nil
}
