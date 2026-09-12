# Docker Compose Guide — dbctl

`dbctl` can automatically generate and manage Docker Compose files for your database infrastructure, or use your own custom compose files. It also supports scaffolding one full, shared database stack (`dbctl init stack`) and auto-discovering it across projects — see [Full Database Stack](#full-database-stack-dbctl-init-stack) and [Host Stack Discovery](#host-stack-discovery) below.

---

## Auto-Generate from Config

`dbctl up` reads your `dbctl.yaml` and generates a complete `docker-compose.dbctl.yml` with all configured database engines, then starts the containers.

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
# dbctl.yaml
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

## Full Database Stack (`dbctl init stack`)

`dbctl up` generates one compose file per **project**, scoped to the drivers in that project's `dbctl.yaml`. `dbctl init stack` is different: it scaffolds one full, standalone Docker Compose stack — MySQL, PostgreSQL, MongoDB, Redis, RabbitMQ, Kafka, NATS, and Typesense — meant to be shared across every project on the machine (or per-repo, your choice).

### Usage

```bash
# ./.dbctl/dbs/docker-compose.yml + ./.dbctl/config.yaml — interactive engine picker in a TTY,
# full stack by default in non-interactive contexts (CI, piped output)
dbctl init stack

# ~/.dbctl/dbs/docker-compose.yml + ~/.dbctl/config.yaml
dbctl init stack global

# ~/.config/dbctl/dbs/docker-compose.yml + ~/.config/dbctl/config.yaml
dbctl init stack config

# A specific directory: <dest>/dbs/docker-compose.yml + <dest>/config.yaml
dbctl init stack /path/to/dest

# Skip the interactive picker
dbctl init stack --engines mysql,postgres,redis

# Regenerate an existing docker-compose.yml (its config.yaml is merged regardless of this flag)
dbctl init stack global --force

# Equivalent shorthand on the base command
dbctl init --stack --engines mysql,redis
```

The generated `config.yaml` (`defaults:` format) wires the driver-backed engines (MySQL, PostgreSQL, MongoDB) to the new compose file automatically:

```yaml
defaults:
  mysql:
    container:
      compose_file: "/home/you/.dbctl/dbs/docker-compose.yml"
      service: "mysql"
  postgres:
    container:
      compose_file: "/home/you/.dbctl/dbs/docker-compose.yml"
      service: "postgres"
```

Redis, RabbitMQ, Kafka, NATS, and Typesense are scaffolded as containers but have no dbctl provisioning driver (no database/user/grant concept) — they're started/stopped like the rest of the stack, just not targets for `dbctl provision`.

### `dbctl init` vs. `dbctl init stack`, and how they interact with `config.yaml`

These two commands write two structurally different kinds of file, kept apart by both name and location so they never collide:

- **`dbctl init [dest]`** scaffolds the **project driver config** (`targets:`/`driver:` format) at `./dbctl.yaml` by default — full host/port/admin credentials, refuses to touch an existing file without `--force`, and `--force` there means "replace the whole file."
- **`dbctl init stack [dest]`** writes a **global-style `defaults:` config** — at `./.dbctl/config.yaml` for the local/default destination, `~/.dbctl/config.yaml` for `global`, etc. — wiring `container.compose_file`/`service` into the driver-backed entries.

For `global`/`project`/`config` destinations, both commands *can* legitimately target the same `defaults:`-format file (e.g. running `dbctl init global` then later `dbctl init stack global`). To avoid one clobbering the other's contributions, `dbctl init stack` never does a blind overwrite of that file:

- If it doesn't exist yet, it's written fresh.
- If it already exists in the global `defaults:` format, only the `container.compose_file`/`service` fields for the selected engines are added or updated — host/port/admin, other drivers, comments, and anything else are left untouched. This happens automatically; `--force` isn't needed and doesn't change this behavior.
- If it exists in a project `targets:`/single-`driver:` format that can't be safely merged, the command errors with the exact `compose_file` path to wire up by hand — it will never guess and overwrite a project config, even with `--force`.

`--force` on `dbctl init stack` only ever controls whether an existing `docker-compose.yml` gets regenerated.

### Stack Engines

| Engine | Service Key | Default Image | Notes |
| :--- | :--- | :--- | :--- |
| MySQL | `mysql` | `mysql:8.0-debian` | Provisioning driver available |
| PostgreSQL | `postgres` | `postgres:16-alpine` | Provisioning driver available |
| MongoDB | `mongo` | `mongo:7` | Provisioning driver available |
| Redis | `redis` | `redis:7-alpine` | Container-only |
| RabbitMQ | `rabbitmq` | `rabbitmq:3-management-alpine` | Container-only |
| Kafka | `kafka` | `confluentinc/cp-kafka:7.6.1` | KRaft single-node mode, container-only |
| NATS | `nats` | `nats:2-alpine` | JetStream enabled, container-only |
| Typesense | `typesense` | `typesense/typesense:27.1` | Container-only |

Each service gets a healthcheck, a persistent named volume (`dbctl-<engine>-data`), and joins the shared `dbctl-net` network described below. The stack's Compose project name is derived from a hash of the stack directory's absolute path (`dbctl-<hash>`), so two stacks generated in differently-located directories that happen to share a folder name (e.g. two unrelated `dbs/` folders) never collide — `docker compose down` in one can't accidentally stop or remove containers from the other.

The per-engine service definitions (images, ports, env vars, healthchecks) live in `pkg/compose/templates/stack/docker-compose.yml`, embedded into the binary at build time — not hardcoded in Go source. Selecting a subset of engines filters that template down to just the chosen services and their volumes.

### Shared Network (`dbctl-net`)

Every service in the generated stack joins a fixed bridge network named `dbctl-net`, so **other, unrelated Docker Compose projects** can reach these databases by service hostname instead of via host-mapped ports — the same pattern used by hand-maintained infrastructure stacks. From another project's compose file:

```yaml
services:
  app:
    image: my-app:latest
    environment:
      DATABASE_URL: "mysql://root:rootpassword@mysql:3306/my_app_db"
    networks:
      - dbctl-net

networks:
  dbctl-net:
    external: true
```

Because the network name is fixed and not tied to the stack directory, only run one canonical stack that owns `dbctl-net` at a time (typically the host-global one at `~/.dbctl/dbs`); other projects should always reference it with `external: true` rather than declaring their own `dbctl-net`.

### Host Stack Discovery

`ResolveComposeFile` (in `pkg/container`) is used by `dbctl up`, `dbctl status`, and per-target container checks to find the compose file to operate on, in this order:

1. An explicit path — e.g. a target's `container.compose_file` in your config (after `${VAR}` expansion)
2. The `DBCTL_COMPOSE_FILE` environment variable (a file path)
3. The `DBCTL_STACK` environment variable (a directory — checked for `docker-compose.yml`/`.yaml`/`compose.yml`/`.yaml`)
4. `./.dbctl/dbs/docker-compose.yml` (project-local)
5. `~/.dbctl/dbs/docker-compose.yml`, `~/.config/dbctl/dbs/docker-compose.yml`
6. `./dbs/docker-compose.yml`, `./docker-compose.yml`, `./compose.yml`, `./compose.yaml`

`dbctl up` uses this resolution (when no `--compose` flag is passed) to prefer an already-discovered stack over generating a fresh `docker-compose.dbctl.yml` from config — so once a host stack exists and `DBCTL_COMPOSE_FILE` is exported, `dbctl up` in any project just starts/reuses it.

---

## Custom Compose Files

If you prefer to maintain your own Docker Compose file, use the `--compose` flag to tell `dbctl up` to use it instead of auto-generating:

```bash
# Use your own compose file
dbctl up --compose ./docker-compose.yml

# Or per-target in dbctl.yaml
```

### Per-Target Container Config

You can configure container settings directly in your `dbctl.yaml` per target:

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
