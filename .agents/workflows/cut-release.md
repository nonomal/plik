---
description: Cut a new Plik release (stable or RC) — changelog, docs, commit, PR, tag, GitHub release
---

# Cut a New Release

Walk through the full release checklist: changelog, documentation, commit, PR, tag, and GitHub release.

CRITICAL RULE: NEVER perform any write action on GitHub without explicit user permission. Always present content for review and wait for explicit approval BEFORE publishing anything.

CRITICAL RULE: Explicitly ask for review and confirmation between EACH step. Do NOT proceed to the next step unless the user has confirmed.

## When to Use

- When the user wants to cut a new release (stable or RC)
- Invoked via `/cut-release`

## Steps

### 0. Gather release information

Ask the user:
1. **Version string** — e.g. `1.4`, `1.4.1`, `1.4-RC1`
2. **Release type** — stable release or release candidate (RC)?

Determine:
- If the version contains `-RC` or similar suffix → RC release
- Otherwise → stable/latest release

This distinction matters for:
- Whether to update `README.md` (stable only)
- Docker tagging (`latest` tag is only for stable — see `releaser/release.sh`)

**⏸️ Wait for user confirmation before proceeding.**

### 1. Run security vulnerability checks

Scan for known vulnerabilities in both the Go dependencies and the frontend:

```bash
make vuln
```

This runs:
- **`govulncheck ./...`** — reports Go modules with known CVEs
- **`npm audit`** (in `webapp/`) — checks npm dependencies for known vulnerabilities

Focus on `high` and `critical` severity — `moderate` and below can be noted but are not necessarily release-blockers.

If vulnerabilities are found, present them to the user and discuss whether to fix, bump, or acknowledge before proceeding.

**Go version bump**: Always check the latest available Go patch version by looking up the [Go downloads page](https://go.dev/dl/) or probing `curl -sI https://go.dev/dl/go<version>.linux-amd64.tar.gz | head -1`. If a newer patch version exists than what `go.mod` currently specifies, propose bumping the `go.mod` Go directive. Include the `go.mod` change in the release commit and use the newer version in the changelog. This ensures the CI Docker image (`golang:1-bookworm`) builds with the latest patched Go version.

**⏸️ Present the vulnerability scan results. Wait for user confirmation before proceeding.**

### 2. Check dependency freshness

Run a dependency audit to identify available updates:

```bash
go list -mod=mod -m -u all 2>&1 | grep '\[v'
```

> [!IMPORTANT]
> The `-mod=mod` flag is required because Plik vendors its dependencies. Without it, `go list -u` silently fails in vendored projects.

Categorize the output:
- **Direct dependencies** — listed in `go.mod` with no `// indirect` comment
- **Indirect dependencies** — transitive deps, lower priority

For each outdated **direct** dependency, check the release notes or changelog for breaking changes or notable behavior changes. Present a summary table of available updates (module, current version, available version, any breaking changes noted) and let the user decide which to bump. After bumping, run `go mod tidy && go mod vendor` and verify the build compiles.

**⏸️ Present the dependency audit summary. Wait for user confirmation before proceeding.**

### 2.5. Check frontend dependency freshness

Run the npm outdated report in the webapp directory:

```bash
cd webapp && npm outdated; true
```

Categorize the output:
- **Semver-safe updates** (`Current` → `Wanted` — same major) — bump these with `npm update`
- **Major version bumps** (`Current` → `Latest` — different major) — investigate breaking changes before deciding

For each **major** version bump, check the project's migration guide or changelog:
- If it's a **drop-in** (e.g., vue-router 4→5 has no breaking changes for standard usage) → bump it
- If it requires **config or code changes** (e.g., vite 7→8 switches to Rolldown bundler) → defer to a dedicated PR

After bumping, run `make test-frontend` to confirm all tests pass:

```bash
make test-frontend
```

**⏸️ Present the frontend dependency audit summary. Wait for user confirmation before proceeding.**

### 3. Check build pipeline versions

Before starting the release, actively check for newer versions of all base images in the `Dockerfile` and propose updates:

| Image | How to check |
|-------|-------------|
| `node:<major>-alpine` | Search for the current Node.js LTS schedule. If a newer LTS major exists, propose updating the Dockerfile. |
| `golang:1-bookworm` | The Go version was already checked in Step 1. Ensure `go.mod` is bumped to the latest patch. |
| `alpine:<version>` | Search for the latest Alpine stable release. If a newer minor/patch exists, propose updating the Dockerfile. |

**All three must be checked every release.** If any updates are available, propose the Dockerfile changes and include them in the release commit. Do not treat this step as informational — outdated base images should be bumped.

Also verify that the `go.mod` Go directive matches the version that `golang:1-bookworm` will resolve to in CI.

> [!TIP]
> The Go version from Step 1 is needed for the changelog ("Binaries will be built with Go X.Y.Z").

**⏸️ Present findings to the user. Wait for confirmation before proceeding.**

### 3.5. Check GitHub Actions versions

Audit all workflow files in `.github/workflows/` for outdated action versions:

```bash
grep -rn 'uses:' .github/workflows/ | grep -v '\./'
```

For each third-party action, check whether a newer major version exists that supports Node.js 24 (GitHub's minimum runtime). Key actions to watch:

| Action | How to check |
|--------|-------------|
| `actions/checkout` | Check [releases](https://github.com/actions/checkout/releases) for latest major |
| `actions/setup-node` | Check [releases](https://github.com/actions/setup-node/releases) for latest major |
| `actions/setup-go` | Check [releases](https://github.com/actions/setup-go/releases) for latest major |
| `actions/upload-artifact` | Check [releases](https://github.com/actions/upload-artifact/releases) for latest major |
| `actions/github-script` | Check [releases](https://github.com/actions/github-script/releases) for latest major |
| `docker/login-action` | Check [releases](https://github.com/docker/login-action/releases) for latest major |
| `docker/setup-buildx-action` | Check [releases](https://github.com/docker/setup-buildx-action/releases) for latest major |
| `softprops/action-gh-release` | Check [releases](https://github.com/softprops/action-gh-release/releases) for latest major |
| `azure/setup-helm` | Check [releases](https://github.com/Azure/setup-helm/releases) for latest major |
| `appleboy/ssh-action` | Check [releases](https://github.com/appleboy/ssh-action/releases) for latest version |
| `peaceiris/actions-gh-pages` | Check [releases](https://github.com/peaceiris/actions-gh-pages/releases) for latest major |

If any actions are outdated, propose updates and include them in the release commit.

**⏸️ Present findings to the user. Wait for confirmation before proceeding.**

### 4. Review documentation

Verify that documentation is up-to-date with the changes in this release:

1. **README.md** — Check that features, examples, and links are current
2. **User-facing docs (`docs/`)** — Review any doc pages related to changed features
3. **AGENTS.md** — Check that agent instructions reflect current state
4. **ARCHITECTURE.md files** — Verify architecture docs match the codebase

To scope the review, look at what changed since the last release:
```bash
git diff <previous-tag>..HEAD --stat -- docs/ README.md AGENTS.md ARCHITECTURE.md
```

Check if any changes warrant documentation updates:
```bash
git log <previous-tag>..HEAD --oneline
```

**⏸️ Present the documentation review findings to the user. If updates are needed and approved by the user, make them, run `make docs` and wait for approval. If everything is up to date, confirm with the user before proceeding.**

### 5. Generate the changelog

Look at `changelog/` for the format convention of existing entries. The format is:

```
Plik <VERSION>

Hi, today we're releasing <description> !

Here is the changelog :

New :
 - Feature description (#issue)

Fix :
 - Bug fix description (@external_contributor)

Documentation :
 - Doc change description

Binaries will be built with Go <version>

Faithfully,
The plik team
```

To build the changelog:
1. Identify the previous release tag: `git describe --tags --abbrev=0`
2. List all commits since the last tag: `git log <previous-tag>..HEAD --oneline`
3. Group changes into categories: New, Fix, Documentation, Misc
4. Include issue/PR/external contributor references where applicable
5. Add any changes from the previous step
6. Add the go version message
7. Write the changelog to `changelog/<VERSION>` (e.g. `changelog/1.4`)

No need to include each and every commit, if one commit is only a small fix or a follow up of another one include only the primary feature/bug.
No need to tag maintainers (@camathieu and @bodji)

For example:

```
New :
 - MCP server for AI assistant integration

Documentation :
 - Add MCP upload example screenshot
```

No need to include `Add MCP upload example screenshot` unless it comes from an external contributor

**⏸️ Present the changelog to the user for review. They may want to edit it. Wait for explicit approval before proceeding.**

### 6. Update the Helm chart changelog

Open `charts/plik/CHANGELOG.md`. Move all content under `[Unreleased]` into a new `[<VERSION>]` heading, and leave `[Unreleased]` empty for future changes:

```diff
 ## [Unreleased]
-
-### Changed
-- item that was unreleased

+## [<VERSION>]
+
+### Changed
+- item that was unreleased
```

If there are no unreleased changes, add a version heading with a note like:
```markdown
## [<VERSION>]

No Helm chart changes in this release.
```

**⏸️ Present the updated Helm CHANGELOG to the user for review. Wait for explicit approval.**

### 7. Update README.md (stable releases only)

**Skip this step entirely for RC releases.**

For stable releases, update the version references in `README.md`:
- The `wget` download URL in the Quick Start section
- The `tar xzvf` command
- The `cd` command
- Any other version-specific references

Search for the previous stable version string and replace with the new version.

**⏸️ Present the README diff to the user for review. Wait for explicit approval.**

### 8. Create the release commit

Stage all changes:
```bash
git add changelog/<VERSION>
git add charts/plik/CHANGELOG.md
git add README.md  # if modified (stable only)
# any other documentation files that were updated
```

Propose a commit message:
```
chore(release): prepare <VERSION>

- Add changelog/<VERSION>
- Update Helm chart CHANGELOG
- Update README.md version references  # if applicable
- Update documentation  # if applicable
```

**⏸️ Present the commit message to the user. Do NOT commit without explicit approval.**

### 9. Create the pull request

1. Create a branch (if not already on one):
   ```bash
   git checkout -b release/<VERSION>
   ```
2. Push the branch:
   ```bash
   git push -u origin release/<VERSION>
   ```
3. Draft a PR targeting `master`:
   - **Title**: `chore(release): prepare <VERSION>`
   - **Body**: Include the change made (Changelog, Chart, Readme, Docs,...)

**⏸️ Present the PR draft to the user. Do NOT create the PR on GitHub without explicit approval.**

### 10. Create the GitHub release

After the PR is merged, create the GitHub release. This creates the tag and the release in a single operation.

> [!IMPORTANT]
> The `release.yaml` GitHub Actions workflow triggers on `release: created`. Creating the release is what kicks off the CI build — it builds release archives, Docker images, packages the Helm chart, and uploads all artifacts to this release. Make sure the PR is merged to `master` first so the tag points to the right commit.

Use the GitHub MCP tools or GH CLI to create a release:
- **Tag**: `<VERSION>` (targeting `master`)
- **Title**: `Plik <VERSION>`
- **Body**: Use the same content as `changelog/<VERSION>`
- **Pre-release**: `true` if RC, `false` if stable
- **Latest**: `true` only if this is a stable release

**⏸️ Present the full release content to the user. Do NOT create the GitHub release without explicit approval.**

### 11. Post-Release Checklist

After the release is published:

- [ ] **Wait for CI** — watch the GitHub Actions `release` workflow until it completes successfully
- [ ] **Pull Docker image** and verify tags exist and point to the right image:
  ```bash
  docker pull rootgg/plik:<VERSION>
  docker pull rootgg/plik:preview          # all releases
  docker pull rootgg/plik:latest           # stable releases only
  ```
- [ ] **Smoke-test the image** — start the server with a default admin and verify `/version`:
  ```bash
  docker run --rm -d -p 8080:8080 --name plik-release-check \
    -e PLIKD_FEATURE_AUTHENTICATION=enabled \
    -e PLIKD_DEFAULT_ADMIN_LOGIN=admin \
    -e PLIKD_DEFAULT_ADMIN_PASSWORD=smoketest \
    rootgg/plik:<VERSION>

  # Unauthenticated — basic version check
  curl -s http://127.0.0.1:8080/version | jq .

  # Authenticated — full build metadata (admin-only)
  curl -s -c /tmp/plik-cookies -X POST http://127.0.0.1:8080/auth/local/login \
    -H 'Content-Type: application/json' \
    -d '{"login":"admin","password":"smoketest"}'
  curl -s -b /tmp/plik-cookies http://127.0.0.1:8080/version | jq .
  ```
  Verify the unauthenticated response:
  - `version` = `<VERSION>`
  - `clients` array is populated (13 entries: bash, darwin, freebsd, linux, openbsd, windows)
  - `releases` array includes the new version as the last entry

  Verify the authenticated (admin) response additionally includes:
  - `isRelease` = `true`
  - `isMint` = `true`
  - `goVersion` = expected Go version (e.g. `go1.26.0 linux/amd64`)
- [ ] **Test client round trip** — while the container is still running, download the CLI client and verify upload/download:
  ```bash
  # Download and make executable
  curl -sf http://127.0.0.1:8080/clients/linux-amd64/plik -o /tmp/plik && chmod +x /tmp/plik

  # Verify client version
  /tmp/plik --version

  # Upload a test file
  echo "release smoke test" > /tmp/plik-smoke-test.txt
  UPLOAD_URL=$(/tmp/plik --server http://127.0.0.1:8080 /tmp/plik-smoke-test.txt 2>&1 | grep -oP 'http://\S+')

  # Download and verify content
  curl -sf "$UPLOAD_URL" -o /tmp/plik-smoke-download.txt
  diff /tmp/plik-smoke-test.txt /tmp/plik-smoke-download.txt && echo "ROUND TRIP OK" || echo "ROUND TRIP FAIL"

  # Cleanup
  rm -f /tmp/plik /tmp/plik-smoke-test.txt /tmp/plik-smoke-download.txt /tmp/plik-cookies
  docker stop plik-release-check
  ```
  Verify:
  - `/tmp/plik --version` outputs `<VERSION>`
  - Upload succeeds and returns a URL
  - Downloaded content matches the original file
- [ ] **Verify Helm repo** — check that the chart index on `gh-pages` includes the new version:
  ```bash
  curl -s https://root-gg.github.io/plik/index.yaml | grep <VERSION>
  ```
- [ ] **Verify Debian packages** — boot a Debian container and test APT repo setup + package install:
  ```bash
  docker run --rm debian:bookworm bash -c '
    set -e
    apt-get update && apt-get install -y curl gnupg
    curl -fsSL https://root-gg.github.io/plik/apt/gpg.key | gpg --dearmor -o /etc/apt/keyrings/plik.gpg
    echo "deb [signed-by=/etc/apt/keyrings/plik.gpg] https://root-gg.github.io/plik/apt stable main" > /etc/apt/sources.list.d/plik.list
    apt-get update
    apt-get install -y plik-server plik-client
    echo "--- Verify versions ---"
    plik --version
    plikd --version
    echo "--- Verify installed files ---"
    dpkg -L plik-server | head -20
    dpkg -L plik-client
    echo "--- Verify systemd unit ---"
    test -f /usr/lib/systemd/system/plikd.service && echo "systemd unit: OK" || echo "systemd unit: MISSING"
    echo "--- All checks passed ---"
  '
  ```
  Verify:
  - Both packages install without errors
  - `plik --version` and `plikd --version` output `<VERSION>`
  - The systemd service unit is installed
- [ ] **Verify GitHub release page** — check that the changelog and release artifacts (archives + Helm chart `.tgz` + `.deb` files) are attached

## Important Notes

- **Never push tags, create PRs, or publish releases without explicit user approval** — this is a hard rule
- **RC releases** do NOT update `README.md` and do NOT get the `latest` Docker tag
- **Stable releases** update `README.md` and get the `latest` Docker tag
- The Helm chart `Chart.yaml` uses `__VERSION__` placeholders — do NOT replace them manually; `helm_release.sh` handles this at build time
- The `release.yaml` workflow handles the actual build, Docker push, Helm packaging, and artifact upload — this workflow only prepares the release metadata