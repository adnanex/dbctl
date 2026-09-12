# Architecture & Overview - dbctl

`dbctl` is a declarative, idempotent database provisioning and container orchestration CLI tool written in Go.

---

## The Problem `dbctl` Solves

In modern development environments:
1. **Developer Setup Friction**: Every engineer or CI job needs databases, users, and specific permissions before backend services can run.
2. **Credential Leaks**: Projects often commit root passwords or connection strings to git repositories just so local dev setups work.
3. **Manual Bootstrap Steps**: Teams maintain complex bash scripts with `docker exec`, `mysql -u root`, and custom `CREATE DATABASE` queries that frequently fail when run multiple times or across different OS environments.
4. **Container State Desync**: Databases aren't running when migration scripts fire, causing connection timeouts and pipeline crashes.

`dbctl` eliminates these headaches with:
- **Declarative YAML Specifications**: State *what* databases, users, and permissions you want.
- **Global Host Defaults (`~/.dbctl/config.yaml`, or project-local `./.dbctl/config.yaml`)**: Define root credentials and ports once. Projects never need root passwords in their repos.
- **Container Awareness**: Auto-detects if the container is stopped, launches it via Docker Compose or standalone `docker run`, and polls port readiness before provisioning.
- **Strict Idempotency**: Run it once, twice, or 100 times — it guarantees the desired state without erroring.

---

## Architectural Diagram

```text
               +----------------------------------+
               |      Developer / CI Pipeline     |
               +----------------------------------+
                                |
                                v
               +----------------------------------+
               |              dbctl               |
               +----------------------------------+
                                |
        +-----------------------+-----------------------+
        |                       |                       |
        v                       v                       v
+----------------+      +----------------+      +----------------+
| Global Config  |      |   Env Files    |      | Project Config |
| ./.dbctl/ or   |      |  .env.mysql    |      |  dbctl.yaml    |
| ~/.dbctl/      |      |  .env          |      |  (minimal)     |
| config.yaml    |      |                |      |                |
+----------------+      +----------------+      +----------------+
        |                       |                       |
        +-----------------------+-----------------------+
                                |  (Merge & Normalize)
                                v
               +----------------------------------+
               |     Target Database Engines      |
               +----------------------------------+
                                |
        +-----------------------+-----------------------+
        | (Check & Ensure Container Running)            |
        v                                               v
+-------------------------+                   +-------------------------+
| Docker Compose Service  |                   | Standalone Docker Run   |
| (e.g. docker-compose.yml|                   | (e.g. mysql:8.0-debian) |
+-------------------------+                   +-------------------------+
                                |
                                v (TCP Port Polling & Readiness)
               +----------------------------------+
               |         Driver Registry          |
               +----------------------------------+
                   |            |            |
                   v            v            v
               [MySQL]      [Postgres]    [MongoDB]
                   |
                   v
         - Check Schema / DB Exists
         - Create Database if missing
         - Check User & Host Exists
         - Create or Synchronize Password (ALTER USER)
         - Grant Privileges (GRANT ... ON ...)
         - Flush Privileges
```

---

## Core Components

1. **Config Engine (`pkg/config`)**:
   - Parses YAML schemas with support for single-target or multi-target definitions.
   - Evaluates environment variable syntax (`${VAR}` and `${VAR:-default}`).
   - Loads cascading `.env` files with precedence.
   - Discovers and merges global host configurations.

2. **Container Orchestrator (`pkg/container`)**:
   - Manages container lifecycles transparently via standard `docker` and `docker compose` CLI calls.
   - Probes network ports with backoff until database processes are accepting incoming client handshakes.
   - `ResolveComposeFile` discovers a host/project-wide compose stack via `DBCTL_COMPOSE_FILE`/`DBCTL_STACK` or well-known paths (`~/.dbctl/dbs`, `./dbs`), so projects don't need their own compose file.

3. **Stack Generator (`pkg/compose`)**:
   - `dbctl up` generates a project-scoped compose file from `dbctl.yaml` targets — or, when reusing a discovered/custom compose file, narrows `docker compose up` to just the service(s) this project's config wires up. Either way, an optional driver-name argument (`dbctl up mysql`) narrows it further to a subset of targets.
   - `dbctl init stack` generates a full, standalone multi-engine stack (MySQL, PostgreSQL, MongoDB, Redis, RabbitMQ, Kafka, NATS, Typesense) by filtering an embedded Compose template (`pkg/compose/templates/stack/`) — engine definitions live in that template, not hardcoded in Go.
   - Every stack it generates gets a hashed, directory-unique Compose project name (avoiding cross-stack `down`/`up` collisions) and joins a fixed shared network (`dbctl-net`) that other, unrelated compose projects can attach to.

4. **Driver Registry (`pkg/driver`)**:
   - Provides a unified `Driver` interface:
     - `Connect(ctx, target)`
     - `Close()`
     - `EnsureDatabase(ctx, dbSpec)`
     - `EnsureUser(ctx, userSpec)`
     - `EnsureGrant(ctx, userSpec, grantSpec)`
   - Pluggable design allowing new engines (Postgres, Mongo, ClickHouse, CockroachDB) to be added without touching the CLI core.
