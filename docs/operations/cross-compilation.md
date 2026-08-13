# Cross Compilation

Plik supports cross-compilation for multiple OS/architecture combinations.

## Building Clients

Build clients for all supported platforms:

```bash
make clients
```

Or build for a specific target:

```bash
TARGETS="linux/amd64" make clients
```

## Supported Platforms

Clients are built for:
- **linux** — amd64, i386, arm, arm64
- **darwin** — amd64, arm64
- **windows** — amd64, i386
- **freebsd** — amd64

The complete list of targets is defined in the `Makefile`.

## Building for Docker

Build a Docker image with the server and all clients:

```bash
make docker
```

The Docker build uses multi-stage builds from `Dockerfile` at the project root.
