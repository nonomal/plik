# Prometheus Metrics

Plik exposes Prometheus metrics on a separate HTTP port.

## Configuration

```toml
MetricsPort = 9090
MetricsAddress = "0.0.0.0"
```

::: tip
`MetricsPort` defaults to `0` (disabled). Set it to a non-zero port to enable metrics.
:::

Metrics are available at `http://localhost:{MetricsPort}/metrics`.

## Available Metrics

### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `plik_http_request_total` | Counter | method, path, code | Count of HTTP requests |

### Server Statistics

| Metric | Type | Description |
|--------|------|-------------|
| `plik_uploads_count` | Gauge | Total uploads in the database |
| `plik_anonymous_uploads_count` | Gauge | Anonymous uploads in the database |
| `plik_files_count` | Gauge | Total files in the database |
| `plik_size_bytes` | Gauge | Total upload size (bytes) |
| `plik_anonymous_size_bytes` | Gauge | Anonymous uploads size (bytes) |
| `plik_users_count` | Gauge | Total users in the database |
| `plik_server_stats_refresh_duration_second` | Histogram | Duration of server stats refresh |
| `plik_last_stats_refresh_timestamp` | Gauge | Timestamp of the last stats refresh |

### Cleaning Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `plik_cleaning_removed_uploads` | Counter | Uploads removed (marked for deletion) |
| `plik_cleaning_deleted_files` | Counter | Files deleted from data backend |
| `plik_cleaning_deleted_uploads` | Counter | Uploads fully deleted |
| `plik_cleaning_removed_orphan_files` | Counter | Orphan files cleaned |
| `plik_cleaning_removed_orphan_tokens` | Counter | Orphan tokens cleaned |
| `plik_cleaning_duration_second` | Histogram | Duration of cleaning runs |
| `plik_last_cleaning_timestamp` | Gauge | Timestamp of the last cleaning |

### Runtime Metrics

Standard Go runtime metrics (goroutines, memory, GC) via `ProcessCollector` and `GoCollector`.
