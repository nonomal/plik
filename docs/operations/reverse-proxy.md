# Reverse Proxy

Plik can be deployed behind a reverse proxy (Nginx, Apache, Caddy, Traefik, etc.).

## Path Prefix

If Plik is served under a sub-path, set the `Path` config parameter:

```toml
Path = "/plik"
```

## Source IP

Behind a proxy, the source IP will be the proxy's address. Configure the source IP header:

```toml
SourceIpHeader = "X-Forwarded-For"
```

## Nginx Example

```nginx
server {
    listen 443 ssl;
    server_name plik.example.com;

    client_max_body_size 10G;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_request_buffering off;
        proxy_buffering off;
        proxy_read_timeout 86400s;
    }
}
```

::: warning
Set `client_max_body_size` to match your `MaxFileSizeStr` setting. Set `proxy_read_timeout` high enough for large uploads (default is 60s, which is too low). For streaming uploads, disable request buffering.
:::

## Apache Example

```apache
<VirtualHost *:443>
    ServerName plik.example.com

    ProxyPreserveHost On
    ProxyPass / http://127.0.0.1:8080/
    ProxyPassReverse / http://127.0.0.1:8080/

    RequestHeader set X-Forwarded-For %{REMOTE_ADDR}s
    RequestHeader set X-Forwarded-Proto https
</VirtualHost>
```

## Caddy Example

```caddyfile
plik.example.com {
    reverse_proxy 127.0.0.1:8080

    request_body {
        max_size 10G
    }
}
```

::: tip
Caddy does not buffer request bodies and has no read timeout by default, making it well-suited for large file uploads out of the box. The only setting to adjust is `max_size` to match your `MaxFileSizeStr` (default is ~100MB).
:::

## Traefik Example

Route Plik via Docker labels on the Plik container:

```yaml
services:
  plik:
    image: rootgg/plik:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.plik.rule=Host(`plik.example.com`)"
      - "traefik.http.routers.plik.entrypoints=websecure"
      - "traefik.http.routers.plik.tls.certresolver=le"
      - "traefik.http.services.plik.loadbalancer.server.port=8080"
    networks:
      - traefik_network
```

::: warning
Traefik's default `readTimeout` is 60 seconds per entrypoint. Large file uploads will fail with a **499 Client Closed Request** error if the upload takes longer than this.

Increase or disable the timeout on your Traefik entrypoints:

```yaml
# In Traefik's docker-compose.yml command section:
- "--entrypoints.websecure.transport.respondingTimeouts.readTimeout=86400"
- "--entrypoints.web.transport.respondingTimeouts.readTimeout=86400"
```

Setting `readTimeout=86400` allows uploads up to 24 hours. This is an **entrypoint-level** setting — Traefik does not support per-route timeouts.
:::

## Disabling Nginx Buffering

By default, Nginx buffers large HTTP requests and responses to a temporary file. This causes unnecessary disk I/O and slower transfers. Disable buffering (requires Nginx ≥1.7.12) for `/file` and `/stream` paths, and increase buffer sizes:

```nginx
proxy_buffering off;
proxy_request_buffering off;
proxy_http_version 1.1;
proxy_buffer_size 1M;
proxy_buffers 8 1M;
client_body_buffer_size 1M;
```

See the [Nginx proxy buffering documentation](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffering) for details.


