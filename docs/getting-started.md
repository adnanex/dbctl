# Getting Started with dbctl

Get up and running with `dbctl` in less than two minutes.

---

## 1. Installation

### From Source
```bash
git clone https://github.com/adnanex/dbctl.git
cd dbctl
make build
```

### Install to System (Global for All Users)
```bash
sudo make install
# Installs binary to /usr/local/bin/dbctl
```

### Install for Current User Only
```bash
make install INSTALL_DIR=$HOME/.local/bin
# (Ensure ~/.local/bin is in your $PATH)
```

Verify installation:
```bash
dbctl --help
```

---

## 2. Initial Setup

### Step A: Initialize Global Host Defaults (Recommended)
Set up your machine's default database credentials once:

```bash
dbctl init global
```

This creates `~/.dbctl/config.yaml` pre-configured with default credentials and connection settings for MySQL, PostgreSQL, and MongoDB:

```yaml
# ~/.dbctl/config.yaml
defaults:
  mysql:
    host: "127.0.0.1"
    port: 3306
    admin:
      username: "root"
      password: "${MYSQL_ROOT_PASSWORD:-rootpassword}"
```

### Step A2 (Alternative): Scaffold a Full Local Database Stack

Instead of relying on per-project containers, `dbctl init stack` generates one shared Docker Compose stack (MySQL, PostgreSQL, MongoDB, Redis, RabbitMQ, Kafka, NATS, Typesense) that any project on the machine can reuse:

```bash
dbctl init stack global
```

This writes `~/.dbctl/dbs/docker-compose.yml` and a matching `~/.dbctl/config.yaml`. Export the compose file path once (e.g. in `~/.bashrc` or `~/.zshrc`):

```bash
export DBCTL_COMPOSE_FILE="$HOME/.dbctl/dbs/docker-compose.yml"
```

Every project's `dbctl up`/`dbctl status` will now discover and reuse this stack automatically — see [Docker Compose Guide → Host Stack Discovery](compose.md#host-stack-discovery).

Prefer defaults scoped to one repo instead of the whole host? Use `dbctl init project` / `dbctl init stack` (no destination) instead of `global` — both write to `./.dbctl/` in the project root, so the config can be committed and shared with the rest of the team.

### Step B: Initialize a Project
In any project repository, run:

```bash
# Generate minimal zero-admin config:
dbctl init minimal
```

This creates a clean `./dbctl.yaml`:

```yaml
driver: mysql

databases:
  - name: my_project_db

users:
  - username: project_user
    password: my_secret_password_123
    databases:
      - my_project_db
```

---

## 3. Running `dbctl`

### Standard Run
```bash
dbctl -config dbctl.yaml
# Or shorthand (dbctl.yaml is also the default, so -c can be omitted):
dbctl -c dbctl.yaml
```

### Dry-Run Mode (Simulation)
Inspect what `dbctl` would do without making changes to the database:
```bash
dbctl -dry-run -c dbctl.yaml
```

### Passing Environment Files
Load runtime variables dynamically:
```bash
dbctl -env .env.mysql -c dbctl.yaml
```
You can pass multiple files; later files take precedence:
```bash
dbctl -env .env.mysql -e .env.secrets -c dbctl.yaml
```
