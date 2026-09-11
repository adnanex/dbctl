# Configuration Reference - dbctl

`dbctl` reads YAML configuration files that describe the desired state of your database engines, databases, users, permissions, and containers.

---

## Configuration Merging Hierarchy

`dbctl` resolves configuration using a 5-tier inheritance model:

1. **Engine Defaults**: Built-in fallbacks (e.g. host `127.0.0.1`, port `3306`, charset `utf8mb4`).
2. **Global Configuration**: Loaded from `~/.dbctl/config.yaml` or `~/.config/dbctl/config.yaml`.
3. **Environment Files**: Loaded via `-env <file>` or `-e <file>` (and auto-loaded `./.env` if present).
4. **OS Environment Variables**: Interpolated into `${VAR}` or `${VAR:-default}`.
5. **Project Config**: Specified via `-config <path>` (overrides everything above).

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
  compose_file: "./docker-compose.yml" # Path to compose file (default: ./docker-compose.yml)
  service: "mysql"                     # Name of the service inside compose file
  wait_timeout: "45s"                  # Maximum duration to wait for port readiness
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
