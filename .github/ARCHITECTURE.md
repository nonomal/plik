# Architecture — GitHub Actions (`.github/`)

> CI/CD workflows and automation for Plik. For system-wide overview, see the root [ARCHITECTURE.md](../ARCHITECTURE.md).

---

## Structure

```
.github/
├── workflows/
│   ├── tests.yaml                 ← CI: lint, test, docs build on PR
│   ├── pages.yml                  ← Deploy docs to gh-pages
│   ├── release.yaml               ← Build release archives, Docker images, Helm chart on release
│   ├── master.yaml                ← Post-merge actions on master
│   ├── docker-build-pr.yaml       ← Build Docker image on PR (triggered by comment)
│   ├── docker-build-reusable.yaml ← Reusable Docker build job (called by build/deploy)
│   └── docker-deploy-pr.yaml      ← Deploy PR image to staging (triggered by comment)
└── ARCHITECTURE.md                ← this file
```

---

## Workflows

### `tests.yaml` — CI Tests

Runs on every pull request. Contains 7 parallel jobs:

| Job | What it does |
|-----|--------------|
| `test` | Go lint + unit tests (`make lint`, `make test`) + uncommitted changes check + `govulncheck` |
| `test-frontend` | Webapp unit tests (`make test-frontend` — vitest) |
| `test-frontend-e2e` | Webapp E2E tests (`make test-frontend-e2e` — Playwright + Chromium) |
| `test-backends` | Docker-based backend integration tests (`make test-backends`) |
| `release` | Release build dry-run (`make release` — Docker buildx) |
| `docs` | VitePress docs build (`make docs` — builds client, injects `--help` + `.plikrc` into docs, fails if out of date) |
| `helm-docs` | Helm chart README regeneration check (`make helm-docs` — fails if out of date) |

### `pages.yml` — GitHub Pages (Docs)

Runs on push to `master` when `docs/**`, any `ARCHITECTURE.md`, or the workflow itself changes. Deploys VitePress documentation to the `gh-pages` branch via `peaceiris/actions-gh-pages`.

> [!NOTE]
> The `keep_files: true` flag preserves the Helm `index.yaml` and APT repository (`apt/`) on `gh-pages` (updated by the release workflow).

### `release.yaml` — Tagged Release Pipeline

Triggered when a GitHub release is created (tag push). Runs the full release pipeline:
1. Builds multi-arch Docker images
2. Builds release archives and client binaries
3. Packages the Helm chart and updates `index.yaml` on `gh-pages` (via `releaser/helm_release.sh`)
4. Builds `.deb` packages and updates the APT repository on `gh-pages` (via `releaser/apt_release.sh`)
5. Uploads all artifacts (including `.tgz`, `.deb` files) to the GitHub release

See [releaser/ARCHITECTURE.md](../releaser/ARCHITECTURE.md) for the build details.

### `master.yaml` — Docker Dev Build

Runs on every push to `master` (only in the `root-gg` org). Builds multi-arch Docker images and pushes `rootgg/plik:dev` to Docker Hub via `make release-and-push-to-docker-hub`. This ensures the `dev` tag always reflects the latest `master` state.

### `docker-build-pr.yaml` — PR Docker Build

Triggered by a `docker build` comment on a PR. Builds a Docker image tagged `rootgg/plik:pr-{number}` and pushes it to Docker Hub. Reports back with a comment.

### `docker-deploy-pr.yaml` — PR Docker Deploy

Triggered by a `docker deploy` comment on a PR. Conditionally builds the Docker image via the reusable workflow (if not already built), then deploys it to the staging instance at `plik.root.gg`. Reports back with a deployment confirmation comment.

---

## Helm Chart Release Flow

```mermaid
graph LR
    Release["GitHub Release created"] --> ReleaseWF["release.yaml workflow"]
    ReleaseWF --> Build["Build archives + Docker"]
    ReleaseWF --> HelmScript["releaser/helm_release.sh"]
    HelmScript --> Package["Package plik-helm-VERSION.tgz"]
    Package --> Assets["GitHub Release assets"]
    HelmScript --> UpdateIndex["Update index.yaml"]
    UpdateIndex --> GHPages["gh-pages branch"]
    GHPages --> HelmRepo["helm repo add plik<br/>https://root-gg.github.io/plik"]
```

The Helm chart version is unified with the app version — `Chart.yaml` `version` and `appVersion` are dynamically set to the release tag by `helm_release.sh` at release time.

Users install the chart via:
```bash
helm repo add plik https://root-gg.github.io/plik
helm install plik plik/plik
```

The chart source lives in `charts/plik/`. See the chart's [README](https://github.com/root-gg/plik/blob/master/charts/plik/README.md) for all configuration options.

## APT Repository Release Flow

```mermaid
graph LR
    Release["GitHub Release created"] --> ReleaseWF["release.yaml workflow"]
    ReleaseWF --> AptScript["releaser/apt_release.sh"]
    AptScript --> Nfpm["nfpm builds .deb packages"]
    Nfpm --> Assets["GitHub Release assets"]
    AptScript --> AptRepo["Update apt/ on gh-pages"]
    AptRepo --> Sign["GPG sign Release"]
    Sign --> GHPages["gh-pages branch"]
    GHPages --> AptInstall["apt install plik-server<br/>https://root-gg.github.io/plik/apt"]
```

Two packages are built per architecture (amd64, i386, arm64, armhf):
- `plik-server` — server binary, config, systemd service, webapp, clients
- `plik-client` — CLI binary

Users install via:
```bash
curl -fsSL https://root-gg.github.io/plik/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/plik.gpg
echo "deb [signed-by=/etc/apt/keyrings/plik.gpg] https://root-gg.github.io/plik/apt stable main" | sudo tee /etc/apt/sources.list.d/plik.list
sudo apt update && sudo apt install plik-server
```

### Pre-release checklist

Before creating a GitHub release:

1. Update the version badge and install snippets in `README.md` to reference the new tag
2. Move `charts/plik/CHANGELOG.md` entries from `[Unreleased]` to the new version heading (e.g. `[1.4-RC4] - 2026-02-21`)
3. Ensure `GPG_PRIVATE_KEY` secret is configured for APT repository signing
4. Ensure all `ARCHITECTURE.md` files reflect any structural changes
