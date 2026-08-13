# Streaming

Stream mode enables direct file transfer from uploader to downloader — nothing is stored on the server.

## How It Works

1. The uploader sends a file via the UI or CLI or API with stream mode enabled
2. The upload request **blocks** until a downloader connects (or a timeout expires)
3. Data flows directly from uploader → server → downloader
4. Once the transfer is complete, both connections close
5. No data is persisted on disk

::: tip
Transfers can be very large and last as long as the connection stay open — the timeout only applies to the initial wait for a downloader, not the transfer itself.
:::

## Usage

### CLI

```bash
plik --stream myfile.txt
```

### API

Create an upload with `stream: true`:

```json
POST /upload
{
    "stream": true,
    "files": [
        { "fileName": "myfile.txt", "fileSize": 12345 }
    ]
}
```

::: tip
For stream mode, you need to know the file ID before the upload starts (since it will block). Pass a `files` array in the upload creation request to get file IDs assigned upfront.
:::

## Timeout

By default, the server waits **5 minutes** for a downloader to connect. If no download starts within that period, the upload is interrupted — but the file returns to a retryable state.

The download link remains valid. Click **Retry** in the UI (or re-POST via the API) to restart the wait.

| Behavior | Description |
|----------|-------------|
| Timeout fires | Upload is interrupted, file can be retried |
| Download starts before timeout | Transfer proceeds normally |
| Timeout set to `0` | No timeout — upload blocks indefinitely |

### Configuration

Set `StreamTimeoutStr` in `plikd.cfg` (or via the `PLIKD_STREAM_TIMEOUT_STR` environment variable):

```toml
# Max wait for a streaming download to start (0 = no timeout)
StreamTimeoutStr = "5m"
```

Accepted duration formats: `30s`, `5m`, `1h`, `1d`, `1w`.

## Multi-Instance Deployment

::: warning Stream mode is stateful
Stream mode is **not stateless**. The uploader request blocks on one Plik instance, so the downloader request **must** be routed to the same instance. Your load balancer must hash on the file ID to route stream requests correctly.
:::

### Nginx with LUA

Here's how to route stream requests to the correct instance using Nginx with LUA scripting. Make sure your Nginx is built with LUA support (the `nginx-extras` Debian package ≥1.7.2 includes it).

```nginx
upstream plik {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
}

upstream stream {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    hash $hash_key;
}

server {
    listen 9000;

    location / {
        set $upstream "";
        set $hash_key "";
        access_by_lua '
            _,_,file_id = string.find(ngx.var.request_uri, "^/stream/[a-zA-Z0-9]+/([a-zA-Z0-9]+)/.*$")
            if file_id == nil then
                ngx.var.upstream = "plik"
            else
                ngx.var.upstream = "stream"
                ngx.var.hash_key = file_id
            end
        ';
        proxy_pass http://$upstream;
    }
}
```

This configuration:
- Routes `/stream/` requests to a consistent upstream based on the file ID hash
- Routes all other requests with standard load balancing
