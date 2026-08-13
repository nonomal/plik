# Import / Export

Plik can export and import all metadata (users, tokens, uploads, files, settings) using the `plikd` CLI. This is useful for **backend migrations** (e.g. SQLite → PostgreSQL), **backups**, and **disaster recovery**.

::: warning
Import/export handles **metadata only** — file data stored in the data backend (filesystem, S3, etc.) is not included. You must separately back up or migrate the data backend contents.
:::

## Export

Dump all metadata to a file:

```bash
plikd export /path/to/export.bin
```

Sample output:

```
Exporting metadata from sqlite3 to /path/to/export.bin
exported 3 users
exported 5 tokens
exported 142 uploads
exported 287 files
exported 1 settings
```

The export includes soft-deleted uploads to preserve foreign key integrity. CLI auth sessions are **not exported** — they are ephemeral and have no value outside the running server.

## Import

Load metadata from a previously exported file:

```bash
plikd import /path/to/export.bin
```

Sample output:

```
Importing metadata from /path/to/export.bin to postgres
imported 3 out of 3 uploads
imported 287 out of 287 files
imported 3 out of 3 users
imported 5 out of 5 tokens
imported 1 out of 1 settings
```

### `--ignore-errors`

By default, import stops on the first error (e.g. a duplicate key). Use `--ignore-errors` to skip problematic records and continue:

```bash
plikd import --ignore-errors /path/to/export.bin
```

Failed records are logged to stdout with details.

## File Format

The export file uses Go's [gob](https://pkg.go.dev/encoding/gob) encoding compressed with [Snappy](https://github.com/golang/snappy). This format is:

- **Architecture-independent** — portable across `amd64`, `arm64`, etc.
- **Compact** — binary encoding + compression keeps files small
- **Streaming** — objects are written sequentially, so memory usage stays constant regardless of database size
- **Go-specific** — the file cannot be read by non-Go tools