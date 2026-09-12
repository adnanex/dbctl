# Configuration Reference - dbctl

`dbctl` reads YAML configuration files that describe the desired state of your database engines, databases, users, permissions, and containers.

There are two distinct kinds of config file, resolved independently and never mixed up by filename:

| | Format | Resolution order | Scaffolded by |
| :--- | :--- | :--- | :--- |
| **Project driver config** | `targets:` (or a single root `driver:`) | `./dbctl.yaml`, `./dbctl.yml`, `./.dbctl/config.yaml`, `./.dbctl/config.yml` — first one found | `dbctl init` / `dbctl init minimal` |
| **Global-style defaults** | `defaults:` | `./.dbctl/config.yaml`, `~/.dbctl/config.yaml`, or `~/.config/dbctl/config.yaml` — first one found | `dbctl init global` / `dbctl init project` / `dbctl init stack` |

The project driver config is deliberately not named the generic `config.yaml` — that name is too likely to already be in use by something else in the project. `./.dbctl/config.yaml` is checked by *both* lookups above but for different YAML keys (`targets:`/`driver:` vs. `defaults:`), so one file at that path can hold either — or both, merged together (see [Configuration Merging Hierarchy](#configuration-merging-hierarchy)) — without conflict.

---

## Configuration Merging Hierarchy

`dbctl` resolves configuration using a 5-tier inheritance model:

1. **Engine Defaults**: Built-in fallbacks (e.g. host `127.0.0.1`, port `3306`, charset `utf8mb4`).
2. **Global Configuration**: Loaded from `./.dbctl/config.yaml` (project-local), `~/.dbctl/config.yaml`, or `~/.config/dbctl/config.yaml` — whichever is found first, in that order.
3. **Environment Files**: Loaded via `-env <file>` or `-e <file>` (and auto-loaded `./.env` if present).
4. **OS Environment Variables**: Interpolated into `${VAR}` or `${VAR:-default}`.
5. **Project Config**: Specified via `-config <path>` (default: first existing of `./dbctl.yaml`, `./dbctl.yml`, `./.dbctl/config.yaml`, `./.dbctl/config.yml`); overrides everything above.

---

## Schema Reference

### Root Structure

A configuration file can be structured either as a **Single Target** or a **Multi-Target list**:

#### Multi-Target Format (`targets`)
```yaml
targets:
  - driver: mysql
    host: 127.0.0.1
    port: 3306
    admin:
      username: root
      password: rootpassword
    databases: [...]
    users: [...]

  - driver: postgres
    host: 127.0.0.1
    port: 5432
    admin:
      username: root
      password: rootpassword
    databases: [...]
    users: [...]
```

#### Single-Target Format (Direct Root)
```yaml
driver: mysql
host: 127.0.0.1
port: 3306
admin:
  username: root
  password: rootpassword
databases: [...]
users: [...]
```

---

## Fields Detail

### Target Fields

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `driver` | string | **Yes** | Database engine name (`mysql`, `postgres`, `mongodb`). |
| `host` | string | No | Server hostname or IP address (default: `127.0.0.1`). |
| `port` | integer| No | Database port (default: engine standard). |
| `admin` | object | No | Credentials for the root / administrative account. |
| `admin.username` | string | No | Admin username (e.g. `root`, `postgres`). |
| `admin.password` | string | No | Admin password. |
| `admin.database` | string | No | Default admin database (e.g. `postgres`). |
| `admin.auth_source`| string | No | Auth source database for MongoDB (default: `admin`). |
| `container` | object | No | Container orchestration configuration. |
| `databases` | array | No | List of databases to ensure exist. |
| `users` | array | No | List of users and their access grants. |

---

### Database Specification (`databases`)

Each database entry can be defined as an object or simply a string name:

```yaml
# Shorthand string list
databases:
  - app_db
  - analytics_db

# Or full object with encoding settings:
databases:
  - name: app_db
    charset: utf8mb4
    collation: utf8mb4_unicode_ci
```

---

### User Specification (`users`)

```yaml
users:
  - username: app_user
    password: "${APP_USER_PASSWORD:-secret123}"
    host: "%"                  # Host mask for MySQL (default: "%")
    
    # Shorthand: Grant ALL PRIVILEGES to specific databases
    databases:
      - app_db

    # Or detailed explicit grants:
    grants:
      - database: app_db
        privileges:
          - "SELECT"
          - "INSERT"
          - "UPDATE"
          - "DELETE"
```

---

### Container Orchestration Specification (`container`)

#### Option 1: Docker Compose Service
```yaml
container:
  compose_file: "./docker-compose.yml" # Path to compose file (optional — see below)
  service: "mysql"                     # Name of the service inside compose file
  wait_timeout: "45s"                  # Maximum duration to wait for port readiness
```

If `compose_file` is omitted (or its `${VAR}` expands to empty), `dbctl` auto-discovers a compose file via `DBCTL_COMPOSE_FILE`/`DBCTL_STACK`, then `./.dbctl/dbs/`, `~/.dbctl/dbs/`, `~/.config/dbctl/dbs/`, or `./dbs/` — see [Host Stack Discovery](compose.md#host-stack-discovery). This is what lets a `~/.dbctl/config.yaml` generated by `dbctl init stack` work across every project on the machine without each one repeating the path:

```yaml
container:
  compose_file: "${DBCTL_COMPOSE_FILE:-/home/you/workspaces/stacks/dbs/docker-compose.yml}"
  service: "mysql"
```

#### Option 2: Standalone Docker Run
```yaml
container:
  image: "mysql:8.0-debian"
  name: "dbs-mysql"
  ports:
    - "3306:3306"
  env:
    MYSQL_ROOT_PASSWORD: "rootpassword"
  volumes:
    - "mysql_data:/var/lib/mysql"
  auto_remove: false
  wait_timeout: "45s"
```
