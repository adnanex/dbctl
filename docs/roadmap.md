# Future Roadmap & Feature Ideas - dbctl

This document outlines planned improvements, architectural extensions, and potential features to expand `dbctl` into a comprehensive developer database control suite.

---

## 1. Schema Migrations & Raw SQL Seeders (`dbctl migrate`)

### Feature Overview
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

## 2. Instant Database Snapshots & Fast Reset (`dbctl snapshot`)

### Feature Overview
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

## 3. Secret Store Integrations

### Feature Overview
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

## 4. Drift Detection (`dbctl diff`)

### Feature Overview
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

## 5. Interactive Terminal UI (TUI) Dashboard (`dbctl ui`)

### Feature Overview
A lightweight, terminal-based dashboard built with `bubbletea` / `lipgloss` showing:
- Real-time container health and port bindings.
- List of databases and their disk sizes.
- Active user accounts and their privileges.
- Interactive buttons to ping connections, reset passwords, or trigger snapshot restores.

---

## 6. Cloud Provider & Remote Cluster Provisioning

### Feature Overview
Extend `dbctl` beyond local Docker to provision databases and users on cloud services:
- **DigitalOcean Managed Databases**: API-driven database and user creation.
- **AWS RDS / Aurora**: Provisioning via IAM or Master user credentials.
- **PlanetScale / Neon / Supabase**: Serverless branching and user creation through provider APIs.

---

## 7. GitHub Action & CI/CD Native Runners

### Feature Overview
Provide a pre-built GitHub Action for CI/CD pipelines:

```yaml
- name: Provision Databases
  uses: adnanex/dbctl-action@v1
  with:
    config: ./config.yaml
    env-file: .env.ci
```

---

## 8. Export Connection Strings (`dbctl export` / `dbctl env`)

### Feature Overview
Quickly output ready-to-use connection strings for application configuration or `.env` files:

```bash
# Print connection string for app_user
dbctl url my_app_db --user app_user
# Output: mysql://app_user:my_secret_pass@127.0.0.1:3306/my_app_db?charset=utf8mb4

# Export to current shell or .env
eval $(dbctl env --user app_user)
```

---

## Summary Priority Matrix

| Phase | Feature | Complexity | Impact |
| :--- | :--- | :--- | :--- |
| **P1** | Complete PostgreSQL & MongoDB Driver Implementations | Medium | High |
| **P1** | Database Schema / SQL Seeders (`schema:` & `seed:`) | Low | High |
| **P2** | Connection URL Generator (`dbctl url`) | Low | Medium |
| **P2** | Instant Snapshots & Restore (`dbctl snapshot`) | Medium | High |
| **P3** | Secret Store Integrations (Vault, 1Password, AWS) | Medium | Medium |
| **P3** | Drift Detection (`dbctl diff`) | High | High |
| **P4** | Interactive Terminal UI (TUI) | Medium | Delight |
