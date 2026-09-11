# Future Roadmap & Feature Ideas — dbctl

This document outlines planned improvements, architectural extensions, and potential features to expand `dbctl` into a comprehensive developer database control suite.

---

## ✅ Completed (v0.2.0)

### Cobra + Viper CLI Framework
Migrated from stdlib `flag` to `spf13/cobra` and `spf13/viper`:
- Nested subcommands (`dbctl provision`, `dbctl init`, `dbctl up`, `dbctl down`, `dbctl status`, `dbctl version`)
- Auto-generated `--help`, shell completions (bash/zsh/fish/powershell)
- Persistent flags inherited by child commands (`--config`, `--env`, `--timeout`, `--dry-run`, `--verbose`, `--quiet`)
- Backward compatible: bare `dbctl` still works as `dbctl provision`

### Charmbracelet Terminal UI
Beautiful terminal output using:
- **lipgloss** — styled headers, badges, status icons
- **charmbracelet/log** — structured key-value logging replacing `log.Printf`
- **huh/spinner** — loading spinners during container startup

### Docker Compose Generation (`dbctl up`)
- Auto-generates `docker-compose.dbctl.yml` from config targets
- Supports MySQL, PostgreSQL, MongoDB, Redis with healthchecks
- `--compose` flag for custom compose files
- `--generate-only` to preview without starting

### Docker Compose Lifecycle (`dbctl down`)
- Stop containers from generated or detected compose files
- `--volumes` flag to remove data volumes

### Status Dashboard (`dbctl status`)
- Table view of all configured targets with container state

---

## Planned Features

### 1. Schema Migrations & Raw SQL Seeders (`dbctl migrate`)

After creating a database and user, applications usually need initial schema tables and seeds.
`dbctl` can execute schema files and raw SQL scripts directly:

```yaml
databases:
  - name: my_app_db
    schema:
      dir: "./migrations"       # Path to directory of .sql files
      engine: "native"          # Or integration with goose/golang-migrate
    seed:
      files:
        - "./seeds/initial_data.sql"
```

**Benefits**:
- Zero additional migration tools needed for simple microservices or staging setups.
- Consistent schema bootstrapping across entire teams.

---

### 2. Instant Database Snapshots & Fast Reset (`dbctl snapshot`)

When testing locally or writing end-to-end tests, developers often corrupt local data and want to reset back to a clean state instantly without re-creating all tables.

```bash
# Save current database state to a local snapshot
dbctl snapshot save clean-dev

# After dirtying data during testing:
dbctl snapshot restore clean-dev

# List saved snapshots
dbctl snapshot list
```

**Implementation**:
- Uses fast compressed dumps (`mysqldump` / `pg_dump` through container exec) or local Docker volume checkpoints.

---

### 3. Secret Store Integrations

Instead of reading credentials from plain `.env` files or hardcoded YAML, allow `dbctl` to pull secrets dynamically from modern secret managers:

```yaml
admin:
  username: "root"
  password: "vault://secret/data/mysql#root_password"
  # Or:
  # password: "aws-secrets://my-app/dev/db#password"
  # password: "op://dev-vault/mysql/password" (1Password CLI)
```

**Supported Providers**:
- HashiCorp Vault
- AWS Secrets Manager / Parameter Store
- 1Password CLI (`op`)
- Doppler / Infisical

---

### 4. Drift Detection (`dbctl diff`)

Check if the real database has drifted from what is declared in `config.yaml` without changing anything:

```bash
dbctl diff -c config.yaml
```

**Output**:
```text
[dbctl] Comparing live database (127.0.0.1:3306) with config.yaml:
  [~] User 'app_user'@'%':
      - Live Privileges:   [SELECT, INSERT]
      - Desired Privileges: [SELECT, INSERT, UPDATE, DELETE]
  [+] Database 'audit_db' is missing in target engine.
```

---

### 5. Interactive TUI Dashboard (`dbctl ui`)

A lightweight, terminal-based dashboard built with `bubbletea` / `lipgloss` showing:
- Real-time container health and port bindings.
- List of databases and their disk sizes.
- Active user accounts and their privileges.
- Interactive buttons to ping connections, reset passwords, or trigger snapshot restores.

---

### 6. Quick Connect (`dbctl connect`)

Open a database client session connected to your configured instances:

```bash
dbctl connect mysql             # Opens mysql CLI
dbctl connect postgres          # Opens psql
dbctl connect mongo             # Opens mongosh
dbctl connect --url mysql       # Print connection URL only
```

---

### 7. Export Connection Strings (`dbctl export`)

Output ready-to-use connection strings for application configuration:

```bash
# Print connection URL
dbctl export url mysql
# Output: mysql://app_user:my_secret_pass@127.0.0.1:3306/my_app_db?charset=utf8mb4

# Export as environment variables
dbctl export env
# Output: export DATABASE_URL=mysql://...

# Export to .env file
dbctl export dotenv

# JSON format for programmatic use
dbctl export json
```

---

### 8. Database Reset (`dbctl reset`)

Drop all databases and users, then re-provision from scratch:

```bash
dbctl reset                     # Interactive confirmation required
dbctl reset --confirm           # Skip confirmation
dbctl reset my_app_db           # Reset specific database only
```

---

### 9. Health Checks (`dbctl health`)

Continuously monitor database connection health:

```bash
dbctl health                    # One-shot health check
dbctl health --watch            # Continuous monitoring (bubbletea TUI)
dbctl health --json             # Machine-readable output for CI
```

---

### 10. Profile System

Define multiple environments in a single config:

```yaml
profiles:
  dev:
    targets: [...]
  test:
    targets: [...]
  staging:
    targets: [...]
```

```bash
dbctl provision --profile test
dbctl up --profile staging
```

---

### 11. Cloud Provider & Remote Cluster Provisioning

Extend `dbctl` beyond local Docker to provision databases and users on cloud services:
- **DigitalOcean Managed Databases**: API-driven database and user creation.
- **AWS RDS / Aurora**: Provisioning via IAM or Master user credentials.
- **PlanetScale / Neon / Supabase**: Serverless branching and user creation through provider APIs.

---

### 12. GitHub Action & CI/CD Native Runners

Provide a pre-built GitHub Action for CI/CD pipelines:

```yaml
- name: Provision Databases
  uses: adnanex/dbctl-action@v1
  with:
    config: ./config.yaml
    env-file: .env.ci
```

---

### 13. Additional Database Engines

| Engine | Priority | Notes |
| :--- | :--- | :--- |
| **Redis** | High | Key-value store, ubiquitous in dev stacks |
| **ClickHouse** | Medium | Analytics, growing fast |
| **CockroachDB** | Medium | Distributed SQL, PostgreSQL-compatible wire protocol |
| **MariaDB** | Low | Can share MySQL driver with minor tweaks |
| **SQLite** | Low | No container needed, file-based |
| **Valkey** | Low | Redis fork, same protocol |

---

## Summary Priority Matrix

| Phase | Feature | Complexity | Impact |
| :--- | :--- | :--- | :--- |
| **P1** | Complete PostgreSQL & MongoDB Driver Implementations | Medium | High |
| **P1** | Database Schema / SQL Seeders (`schema:` & `seed:`) | Low | High |
| **P1** | Quick Connect (`dbctl connect`) | Low | Medium |
| **P2** | Connection URL Generator (`dbctl export`) | Low | Medium |
| **P2** | Instant Snapshots & Restore (`dbctl snapshot`) | Medium | High |
| **P2** | Health Checks (`dbctl health`) | Low | Medium |
| **P2** | Database Reset (`dbctl reset`) | Low | Medium |
| **P3** | Secret Store Integrations (Vault, 1Password, AWS) | Medium | Medium |
| **P3** | Drift Detection (`dbctl diff`) | High | High |
| **P3** | Profile System | Medium | Medium |
| **P4** | Interactive TUI Dashboard (`dbctl ui`) | Medium | Delight |
| **P4** | Cloud Provider Provisioning | High | High |
| **P4** | GitHub Action | Medium | Medium |
