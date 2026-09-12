# dbctl

<div align="center">

**Declarative, idempotent database & user management with container orchestration**

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.3.0-brightgreen.svg)]()

*MySQL · PostgreSQL · MongoDB · Redis · RabbitMQ · Kafka · NATS · Typesense*

</div>

---

## What is `dbctl`?

`dbctl` is a command-line tool that automates creating databases, configuring users, managing passwords, and granting permissions declaratively from YAML files.

Instead of writing custom shell scripts or executing brittle `mysql -u root` one-liners, define your desired database state in YAML and run `dbctl`. It ensures idempotent convergence every time.

### Core Highlights

- **Zero-Admin Project Configs**: Set host defaults once in `~/.dbctl/config.yaml`. Project repositories never need to check in root passwords.
- **Full Database Stack Scaffolding**: `dbctl init stack` generates a complete multi-engine `docker-compose.yml` (MySQL, PostgreSQL, MongoDB, Redis, RabbitMQ, Kafka, NATS, Typesense) with healthchecks, persistent volumes, and a shared Docker network other projects can join.
- **Host Stack Discovery**: Any project on the machine auto-discovers a shared host-wide database stack via `DBCTL_COMPOSE_FILE` / `DBCTL_STACK` or well-known paths (`~/.dbctl/dbs`, `./dbs`) — no per-project compose file needed.
- **Docker Compose Generation**: Automatically generates `docker-compose.yml` from your config and starts containers with `dbctl up`.
- **Custom Compose Support**: Bring your own `docker-compose.yml` with `dbctl up --compose ./my-file.yml`.
- **Container Orchestration**: Automatically starts stopped database services via Docker Compose or standalone `docker run` and polls port readiness before provisioning.
- **Strict Idempotency**: Safe to run repeatedly in development, test suites, and CI/CD pipelines.
- **Beautiful Terminal Output**: Powered by [Charmbracelet](https://github.com/charmbracelet) libraries (lipgloss, log, huh) for styled, structured logging.
- **Template Scaffolding**: Run `dbctl init` to generate templates for global defaults, project configs, or minimal zero-admin setups.
- **Multi-Driver Engine**: Pluggable drivers for MySQL, PostgreSQL, and MongoDB (MySQL fully implemented out of the box).
- **Environment Interpolation**: Supports `${VAR}` and `${VAR:-default}` syntax and loads multiple cascading `.env` files.
- **Shell Completions**: Auto-generated completions for bash, zsh, fish, and PowerShell.

---

## Quick Start

### 1. Installation

```bash
# Build and install to ~/.local/bin/dbctl
make install INSTALL_DIR=$HOME/.local/bin

# Or install system-wide for all users
sudo make install
```

Verify:
```bash
dbctl version
```

### 2. Configure Host Defaults (One-Time Setup)

```bash
dbctl init global
```

This generates `~/.dbctl/config.yaml`:

```yaml
defaults:
  mysql:
    host: 127.0.0.1
    port: 3306
    admin:
      username: root
      password: "${MYSQL_ROOT_PASSWORD:-rootpassword}"
    container:
      compose_file: "./docker-compose.yml"
      service: "mysql"
```

### 3. Or: Scaffold a Full Database Stack

Instead of (or alongside) per-project containers, generate one shared stack for the whole machine:

```bash
dbctl init stack global
```

This writes `~/.dbctl/dbs/docker-compose.yml` (MySQL, PostgreSQL, MongoDB, Redis, RabbitMQ, Kafka, NATS, Typesense — pick a subset with `--engines mysql,redis`, or answer the interactive prompt) plus a matching `~/.dbctl/config.yaml`. Export the compose file once in your shell profile:

```bash
export DBCTL_COMPOSE_FILE="$HOME/.dbctl/dbs/docker-compose.yml"
```

...and every project on the machine picks it up automatically — `dbctl up`/`dbctl status` discover it without any per-project config. See [Docker Compose Guide](docs/compose.md) for the full discovery order and how other (non-dbctl) compose projects can join the stack's shared network.

### 4. Initialize a Project

In any project repository, generate a minimal configuration:

```bash
dbctl init minimal
```

This creates `./config.yaml`:

```yaml
driver: mysql

databases:
  - name: my_app_db

users:
  - username: my_app_user
    password: "${APP_USER_PASSWORD:-secret_password_123}"
    databases:
      - my_app_db
```

### 5. Start Containers & Provision

```bash
# Option A: Auto-generate Docker Compose + start + provision
dbctl up
dbctl provision

# Option B: Use your own Docker Compose file
dbctl up --compose ./docker-compose.yml
dbctl provision
```

Output:
```text
INFO dbctl: Loading global configuration path=~/.dbctl/config.yaml
INFO dbctl: Loaded driver defaults from global config count=3
INFO dbctl: Loading project configuration path=config.yaml

── [1/1]  mysql  on 127.0.0.1:3306
INFO dbctl: Connecting to database driver=mysql host=127.0.0.1 port=3306 admin=root
INFO dbctl: Database created driver=mysql database=my_app_db charset=utf8mb4
INFO dbctl: User created driver=mysql user=my_app_user host=%
INFO dbctl: Privileges granted driver=mysql target=`my_app_db`.* user=my_app_user
INFO dbctl: Successfully provisioned all resources driver=mysql host=127.0.0.1 port=3306

INFO dbctl: ✓ Successfully provisioned 1/1 database target(s)
```

---

## CLI Reference

### Commands

| Command | Description |
| :--- | :--- |
| `dbctl provision` | Provision databases, users, and permissions from config |
| `dbctl up` | Generate Docker Compose and start database containers |
| `dbctl down` | Stop database containers |
| `dbctl init [dest]` | Scaffold configuration file (`global`, `minimal`, `local`, `<path>`) |
| `dbctl init stack [dest]` | Scaffold a full database stack (`docker-compose.yml` + matching config) |
| `dbctl status` | Show status of configured database targets |
| `dbctl version` | Print version, commit, build date, and Go info |
| `dbctl completion` | Generate shell completion scripts (bash/zsh/fish/powershell) |

### Global Flags

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--config <path>` | `-c` | `config.yaml` | Path to YAML configuration file |
| `--global-config <path>` | — | Auto-detected | Path to global host configuration (`~/.dbctl/config.yaml`) |
| `--env <path>` | `-e` | — | Path to `.env` file (can be repeated) |
| `--skip-container` | — | `false` | Skip automatic container checking and startup |
| `--dry-run` | — | `false` | Simulate execution without making database modifications |
| `--timeout <duration>` | `-t` | `60s` | Overall timeout for provisioning operations |
| `--verbose` | `-v` | `false` | Enable verbose debug output |
| `--quiet` | `-q` | `false` | Suppress all output except errors |

### Docker Compose Commands

```bash
# Generate compose file from config and start containers
dbctl up

# Generate without starting
dbctl up --generate-only

# Use a custom compose file
dbctl up --compose ./my-docker-compose.yml

# Stop containers
dbctl down

# Stop and remove data volumes
dbctl down --volumes
```

`dbctl up` uses a `--compose` file if given; otherwise it auto-discovers an existing host/project stack (see below) before falling back to generating from config.

### Full Database Stack Commands

```bash
# Scaffold ./dbs/docker-compose.yml + ./config.yaml (interactive engine picker in a TTY)
dbctl init stack

# Scaffold at ~/.dbctl/dbs/docker-compose.yml + ~/.dbctl/config.yaml
dbctl init stack global

# Non-interactive, specific engines only
dbctl init stack --engines mysql,postgres,redis
```

### Host Stack Discovery

`dbctl` resolves which Docker Compose file to use (for `up`, `down`, `status`, and per-target container checks) in this order:

1. An explicitly configured `container.compose_file` in `config.yaml`
2. The `DBCTL_COMPOSE_FILE` (file) or `DBCTL_STACK` (directory) environment variables
3. `~/.dbctl/dbs/docker-compose.yml`, `~/.config/dbctl/dbs/docker-compose.yml`
4. `./dbs/docker-compose.yml`, `./docker-compose.yml`

This means any project on a machine with `DBCTL_COMPOSE_FILE` exported (or a stack scaffolded at one of the well-known paths above) transparently reuses that shared stack.

---

## Documentation

Detailed documentation is available in the [`docs/`](docs/) directory:

- [Architecture & Overview](docs/overview.md)
- [Getting Started Guide](docs/getting-started.md)
- [Configuration Reference](docs/configuration.md)
- [Docker Compose Guide](docs/compose.md)
- [Supported Database Engines](docs/engines.md)
- [Future Roadmap & Planned Features](docs/roadmap.md)

---

## Technology Stack

- **CLI Framework**: [Cobra](https://github.com/spf13/cobra) — subcommands, flags, completions, help generation
- **Configuration**: [Viper](https://github.com/spf13/viper) — YAML/env/flag merging and auto-discovery
- **Terminal Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss) — beautiful styled terminal output
- **Structured Logging**: [Charmbracelet Log](https://github.com/charmbracelet/log) — colored, structured logging
- **Spinners & Progress**: [Huh Spinner](https://github.com/charmbracelet/huh) — elegant loading indicators

---

## License

[MIT](LICENSE) © 2026 Adnanex
