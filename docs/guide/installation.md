# Installation

## From Release

Download the latest release and run:

```bash
wget https://github.com/root-gg/plik/releases/download/__VERSION__/plik-server-__VERSION__-linux-amd64.tar.gz
tar xzvf plik-server-__VERSION__-linux-amd64.tar.gz
cd plik-server-__VERSION__/server
./plikd
```

Plik is now running at [http://127.0.0.1:8080](http://127.0.0.1:8080).

Edit `plikd.cfg` to adjust the configuration (ports, TLS, TTL, backends, etc.).

::: tip
Standalone client binaries for all platforms are also available on the [release page](https://github.com/root-gg/plik/releases). See the [CLI documentation](../features/cli-client.md) for installation instructions.
:::

## From Source

Requires Go and Node.js:

```bash
git clone https://github.com/root-gg/plik.git
cd plik && make
cd server && ./plikd
```

## Docker

Plik provides multiarch Docker images for linux amd64/i386/arm/arm64:

```bash
docker run -p 8080:8080 rootgg/plik
```

Available tags:
- `rootgg/plik:latest` — latest release
- `rootgg/plik:{version}` — specific release
- `rootgg/plik:dev` — latest master commit

See the [Docker Deployment Guide](./docker.md) for custom configuration, persistent storage, and Docker Compose examples.

## Debian / Ubuntu

Plik provides `.deb` packages for amd64, arm64, armhf, and i386 via an APT repository hosted on GitHub Pages.

```bash
# Add the repository
curl -fsSL https://root-gg.github.io/plik/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/plik.gpg
echo "deb [signed-by=/etc/apt/keyrings/plik.gpg] https://root-gg.github.io/plik/apt stable main" | sudo tee /etc/apt/sources.list.d/plik.list
sudo apt update

# Install server
sudo apt install plik-server
sudo systemctl start plikd

# Or install just the CLI client
sudo apt install plik-client
```

The server package installs:
- `/usr/bin/plikd` — server binary
- `/etc/plik/plikd.cfg` — configuration (preserved on upgrade)
- Systemd service (`plikd.service`)
- Webapp, client binaries, and changelog under `/usr/share/plik/`
- Data directory at `/var/lib/plik/`

::: tip
The `plik-server` package creates a `plik` system user and automatically adjusts the default configuration paths for the Debian filesystem layout.
:::
