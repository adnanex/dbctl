# Docker Compose Guide — dbctl

`dbctl` can automatically generate and manage Docker Compose files for your database infrastructure, or use your own custom compose files.

---

## Auto-Generate from Config

`dbctl up` reads your `config.yaml` and generates a complete `docker-compose.dbctl.yml` with all configured database engines, then starts the containers.

### Basic Usage

```bash
# Generate docker-compose.dbctl.yml and start containers
dbctl up

# Generate without starting containers
dbctl up --generate-only

# Specify a different config file
dbctl up -c my-project.yaml

# Custom output path for generated compose file
dbctl up --output ./infra/docker-compose.yml
```

### What Gets Generated

For a config with MySQL and PostgreSQL targets:

```yaml
# config.yaml
targets:
  - driver: mysql
    port: 3306
    databases:
      - name: my_app
  - driver: postgres
    port: 5432
    databases:
      - name: analytics
```

`dbctl up --generate-only` produces:

```yaml
# docker-compose.dbctl.yml (auto-generated)
services:
  dbctl-mysql:
    image: mysql:8.0-debian
    container_name: dbctl-mysql
    restart: unless-stopped
    ports:
      - "3306:3306"
    environment:
      MYSQL_ROOT_PASSWORD: "${MYSQL_ROOT_PASSWORD:-rootpassword}"
    volumes:
      - dbctl-mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s

  dbctl-postgres:
    image: postgres:16-alpine
    container_name: dbctl-postgres
    restart: unless-stopped
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD:-rootpassword}"
    volumes:
      - dbctl-postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s

volumes:
  dbctl-mysql-data:
  dbctl-postgres-data:
```

### Supported Engines

| Engine | Default Image | Healthcheck |
| :--- | :--- | :--- |
| MySQL | `mysql:8.0-debian` | `mysqladmin ping` |
| PostgreSQL | `postgres:16-alpine` | `pg_isready` |
| MongoDB | `mongo:7` | `mongosh ping` |
| Redis | `redis:7-alpine` | `redis-cli ping` |

---

## Custom Compose Files

If you prefer to maintain your own Docker Compose file, use the `--compose` flag to tell `dbctl up` to use it instead of auto-generating:

```bash
# Use your own compose file
dbctl up --compose ./docker-compose.yml

# Or per-target in config.yaml
```

### Per-Target Container Config

You can configure container settings directly in your `config.yaml` per target:

```yaml
targets:
  - driver: mysql
    host: 127.0.0.1
    port: 3306
    admin:
      username: root
      password: rootpassword
    container:
      compose_file: "./docker-compose.yml"   # Path to your compose file
      service: "my-custom-mysql"             # Service name in compose file
    databases:
      - name: my_app
```

Or use standalone Docker (no compose):

```yaml
targets:
  - driver: mysql
    container:
      image: "mysql:8.0-debian"
      name: "my-mysql-container"
      ports:
        - "3306:3306"
      env:
        MYSQL_ROOT_PASSWORD: "rootpassword"
      volumes:
        - "mysql_data:/var/lib/mysql"
```

---

## Stopping Containers

```bash
# Stop all managed containers
dbctl down

# Stop and remove data volumes (DESTROYS DATA)
dbctl down --volumes
```

---

## Typical Workflow

```bash
# 1. Initialize project config
dbctl init minimal

# 2. Start database containers
dbctl up

# 3. Provision databases, users, and permissions
dbctl provision

# 4. Check everything is running
dbctl status

# 5. When done, stop containers
dbctl down
```
