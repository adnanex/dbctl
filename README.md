# dbctl

<div align="center">

**Declarative, idempotent database & user provisioner with container orchestration**

[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.1.0-brightgreen.svg)]()

*MySQL · PostgreSQL · MongoDB*

</div>

---

## What is `dbctl`?

`dbctl` is a command-line tool that automates creating databases, configuring users, managing passwords, and granting permissions declaratively from YAML files.

Instead of writing custom shell scripts or executing brittle `mysql -u root` one-liners, define your desired database state in YAML and run `dbctl`. It ensures idempotent convergence every time.

### Core Highlights

- **Zero-Admin Project Configs**: Set host defaults once in `~/.dbctl/config.yaml`. Project repositories never need to check in root passwords.
- **Container Orchestration**: Automatically starts stopped database services via Docker Compose or standalone `docker run` and polls port readiness before provisioning.
- **Strict Idempotency**: Safe to run repeatedly in development, test suites, and CI/CD pipelines.
- **Template Scaffolding**: Run `dbctl init` to generate templates for global defaults, project configs, or minimal zero-admin setups.
- **Multi-Driver Engine**: Pluggable drivers for MySQL, PostgreSQL, and MongoDB (MySQL fully implemented out of the box).
- **Environment Interpolation**: Supports `${VAR}` and `${VAR:-default}` syntax and loads multiple cascading `.env` files.

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
dbctl --help
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

### 3. Initialize a Project

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

### 4. Run `dbctl`

```bash
dbctl -config config.yaml
```

Output:
```text
[dbctl] Loading global configuration from ~/.dbctl/config.yaml...
[dbctl] Loaded 3 driver default(s) from global config
[dbctl] Loading configuration from config.yaml...
--- [1/1] Processing mysql on 127.0.0.1:3306 ---
[dbctl] [container] Docker Compose service "mysql" is already running.
[dbctl] [container] 127.0.0.1:3306 is reachable and ready!
[dbctl] [mysql] Connecting to 127.0.0.1:3306 as root...
[dbctl] [mysql] Successfully created database my_app_db
[dbctl] [mysql] Successfully created user my_app_user@%
[dbctl] [mysql] Successfully granted privileges on `my_app_db`.* to my_app_user@%
[dbctl] Done! Successfully provisioned 1/1 database target(s).
```

---

## Documentation

Detailed documentation is available in the [`docs/`](docs/) directory:

- [Architecture & Overview](docs/overview.md)
- [Getting Started Guide](docs/getting-started.md)
- [Configuration Reference](docs/configuration.md)
- [Supported Database Engines](docs/engines.md)
- [Future Roadmap & Planned Features](docs/roadmap.md)

---

## CLI Flags

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `init [dest]` | — | — | Scaffold configuration file (`global`, `config`, `local`, `minimal`, `<path>`) |
| `-config <path>` | `-c` | `config.yaml` | Path to YAML configuration file |
| `-global-config <path>` | `-g` | Auto-detected | Path to global host configuration (`~/.dbctl/config.yaml`) |
| `-env <path>` | `-e` | — | Path to `.env` file (can be repeated or comma-separated) |
| `-skip-container` | — | `false` | Skip automatic container checking and startup |
| `-dry-run` | — | `false` | Simulate execution without making database modifications |
| `-timeout <duration>` | — | `60s` | Overall timeout for provisioning operations |

---

## License

[MIT](LICENSE) © 2026 Adnanex
