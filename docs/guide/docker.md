# Docker Deployment

Plik provides multiarch Docker images for production deployments.

## Quick Start

```bash
docker run -p 8080:8080 rootgg/plik
```

Available tags:
- `latest` — Latest stable release
- `preview` — Latest release (including pre-releases like `-RC`)
- `__VERSION__` — Specific version (e.g. `1.4.0`, `1.4-RC1`)
- `dev` — Latest build from master branch

## Custom Configuration

Mount a custom `plikd.cfg` file:

```bash
docker run -p 8080:8080 \
  -v /path/to/plikd.cfg:/home/plik/server/plikd.cfg \
  rootgg/plik
```

> [!IMPORTANT]
> All paths in `plikd.cfg` must reference locations **inside the container**, not on the host.

See the [Configuration Guide](./configuration.md) for all available options.

## Persistent Storage

By default, data is stored inside the container and lost when it is removed. To persist data you need two things:

1. **A volume** mounted to a path like `/data`
2. **A custom `plikd.cfg`** that points to that path (the default config writes to the container's local filesystem)

```toml
DataBackend = "file"
[DataBackendConfig]
    Directory = "/data/files"

[MetadataBackendConfig]
    Driver = "sqlite3"
    ConnectionString = "/data/plik.db"
```

### Using a Named Volume (recommended)

Named volumes are managed by Docker. File ownership is set automatically — no extra steps needed.

```bash
docker run -p 8080:8080 \
  -v /path/to/plikd.cfg:/home/plik/server/plikd.cfg \
  -v plik-data:/data \
  rootgg/plik
```

### Using a Bind Mount

Bind mounts map a specific host path into the container, which is useful when you need direct access to the files (backups, NFS, etc.).

Because the Plik image runs as a non-root user **`plik`** (UID=1000 / GID=1000), the host directory must be writable by that UID:

```bash
mkdir -p /data/plik
chown -R 1000:1000 /data/plik

docker run -p 8080:8080 \
  -v /data/plik:/data \
  rootgg/plik
```

## Docker Compose

```yaml
services:
  plik:
    image: rootgg/plik:latest
    container_name: plik
    volumes:
      - ./plikd.cfg:/home/plik/server/plikd.cfg
      - plik-data:/data
    ports:
      - 8080:8080
    restart: "unless-stopped"

volumes:
  plik-data:
```

```bash
docker compose up -d
```

## Environment Variables

You can override config values via environment variables:

```bash
docker run -p 8080:8080 \
  -e PLIKD_MAX_FILE_SIZE_STR=10GB \
  -e PLIKD_DEFAULT_TTL_STR=86400 \
  rootgg/plik
```

## Health Check

```bash
curl http://localhost:8080/health
```

## Verify Your Setup

After deploying, confirm that your data persists across container restarts:

1. Upload a file through the web UI.
2. Restart the container (`docker compose restart plik`).
3. Confirm the file is still accessible.

This catches misconfigured volume mounts or permission issues before they matter in production.

## Building Custom Images

```bash
make docker
```

This creates a `rootgg/plik:dev` image with the current codebase.