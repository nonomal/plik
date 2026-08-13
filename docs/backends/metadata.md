# Metadata Backends

Plik uses GORM for metadata storage, supporting multiple SQL databases.

## SQLite3 (Default)

Best for standalone deployments.

```toml
[MetadataBackendConfig]
    Driver = "sqlite3"
    ConnectionString = "plik.db"
    Debug = false
```

SQLite3 is configured with WAL mode and foreign keys enabled for optimal performance and data integrity.

## PostgreSQL

Best for distributed or high-availability deployments.

```toml
[MetadataBackendConfig]
    Driver = "postgres"
    ConnectionString = "host=localhost user=plik password=plik dbname=plik port=5432 sslmode=disable"
    Debug = false
```

## MySQL / MariaDB

Also suitable for distributed deployments.

```toml
[MetadataBackendConfig]
    Driver = "mysql"
    ConnectionString = "plik:plik@tcp(localhost:3306)/plik?charset=utf8mb4&parseTime=True"
    Debug = false
```

## Connection Pool

For PostgreSQL and MySQL, you can tune the connection pool:

```toml
[MetadataBackendConfig]
    MaxOpenConns = 25
    MaxIdleConns = 10
```

## Slow Query Logging

Enable slow query detection:

```toml
[MetadataBackendConfig]
    SlowQueryThreshold = "200ms"
```

## Schema Migrations

Plik uses [gormigrate](https://github.com/go-gormigrate/gormigrate) for automatic schema migrations. The database schema is created or updated automatically on server start.

## Migrating Between Backends

To migrate data between different metadata backends (e.g. SQLite → PostgreSQL), use the `plikd export` and `plikd import` commands. See the [Import / Export](/operations/import-export) guide for details.
