# Supported Database Engines - dbctl

`dbctl` features a pluggable driver architecture allowing uniform declarations across distinct SQL and NoSQL engines.

This page covers **provisioning drivers** — engines `dbctl provision` can create databases, users, and grants for. Engines below are separate from the **stack engines** available via `dbctl init stack` (MySQL, PostgreSQL, MongoDB, Redis, RabbitMQ, Kafka, NATS, Typesense — see [Docker Compose Guide](compose.md#full-database-stack-dbctl-init-stack)): Redis, RabbitMQ, Kafka, NATS, and Typesense have no database/user/grant concept, so `dbctl` can start/stop them as containers but has no provisioning driver for them.

---

## 1. MySQL

**Status**: Fully Implemented & Production-Ready  
**Driver Identifier**: `mysql`

### Behavior & Guarantees
- **Connection**: Connects via `github.com/go-sql-driver/mysql` with multi-statement support and connection retry backoff.
- **Database Creation**: Checks `INFORMATION_SCHEMA.SCHEMATA` before running `CREATE DATABASE IF NOT EXISTS \`dbname\` CHARACTER SET ... COLLATE ...`.
- **User Creation**: Checks `mysql.user` for `username` and `host` (defaults to `%`). If missing, creates with `CREATE USER ... IDENTIFIED BY ...`. If present, calls `ALTER USER ... IDENTIFIED BY ...` to synchronize credentials.
- **Grants**: Applies `GRANT <privileges> ON \`dbname\`.* TO 'user'@'host'` and invokes `FLUSH PRIVILEGES`.

---

## 2. PostgreSQL

**Status**: Scaffolded / Ready for Implementation  
**Driver Identifier**: `postgres` or `postgresql`

### Planned Implementation (`pkg/driver/postgres`)
- **Connection Driver**: `github.com/jackc/pgx/v5` or `github.com/lib/pq`
- **Database Check**: `SELECT 1 FROM pg_database WHERE datname = $1;`
  - Create: `CREATE DATABASE "dbname" ENCODING 'UTF8';`
- **User Check**: `SELECT 1 FROM pg_roles WHERE rolname = $1;`
  - Create: `CREATE ROLE "user" WITH LOGIN PASSWORD 'password';`
  - Update: `ALTER ROLE "user" WITH PASSWORD 'password';`
- **Grants**: `GRANT ALL PRIVILEGES ON DATABASE "dbname" TO "user";`

---

## 3. MongoDB

**Status**: Scaffolded / Ready for Implementation  
**Driver Identifier**: `mongodb` or `mongo`

### Planned Implementation (`pkg/driver/mongodb`)
- **Connection Driver**: `go.mongodb.org/mongo-driver/mongo`
- **Database & User Provisioning**:
  - Connects to admin auth source (`mongodb://root:password@host:port/admin`).
  - Database check / namespace creation.
  - User check: `admin.runCommand({ usersInfo: "username" })`.
  - Create user: `admin.runCommand({ createUser: "username", pwd: "password", roles: [{ role: "readWrite", db: "dbname" }] })`.
  - Update user: `admin.runCommand({ updateUser: "username", pwd: "password", roles: [...] })`.
