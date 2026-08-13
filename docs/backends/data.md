# Data Backends

Plik supports multiple storage backends for uploaded files.

## File (Default)

Stores files in a local or mounted filesystem directory.

```toml
DataBackend = "file"
[DataBackendConfig]
    Directory = "files"
```

## Amazon S3

Compatible with any S3-compatible storage (AWS, MinIO, etc.).

```toml
DataBackend = "s3"
[DataBackendConfig]
    Endpoint = "s3.amazonaws.com"
    AccessKeyID = "your-access-key"
    SecretAccessKey = "your-secret-key"
    Bucket = "plik"
    Location = "us-east-1"
    Prefix = ""
    UseSSL = true
    BucketLookup = "auto"  # Bucket addressing style: "auto" (default), "dns" (virtual-hosted), "path" (required for Cloudflare R2 and some MinIO deployments)
    PartSize = 16777216  # 16MiB chunks (min 5MiB, max file = PartSize × 10000)
    PartUploadConcurrency = 4  # Parallel part upload threads (default 4, 1 = sequential)
    SendContentMd5 = false  # Use Content-MD5 instead of x-amz-checksum-* headers (for strict S3-compatible APIs like B2)
```

### Provider-Specific Configuration

#### [Cloudflare R2](https://developers.cloudflare.com/r2/)

R2 requires path-style bucket addressing (`BucketLookup = "path"`) because its TLS wildcard certificate does not cover virtual-hosted-style subdomains.  
Set `Location` to `"auto"` or the value defined for your bucket in the R2 Console:

```toml
BucketLookup = "path"
Location = "auto"
```

#### [Backblaze B2](https://www.backblaze.com/cloud-storage)

B2's S3-compatible API does not support the newer `x-amz-checksum-*` headers. Enable legacy Content-MD5 checksums:

```toml
SendContentMd5 = true
```

### Upload Strategy

Plik uses a **buffer-then-decide** strategy for S3 uploads:

- **Small files** (≤ `PartSize`): uploaded via a single PUT request with the exact size — optimal latency, minimal overhead.
- **Large files** (> `PartSize`): uploaded via S3 multipart upload with `PartUploadConcurrency` parallel part uploads (default: 4). Set to 1 for sequential uploads. Memory usage per upload: `PartUploadConcurrency × PartSize`.

### Bucket Versioning

> [!WARNING]
> Plik permanently deletes files from the S3 bucket, even if bucket versioning is enabled. Consider disabling versioning on your Plik bucket to avoid accumulating unnecessary delete markers.

### Server-Side Encryption

| Mode | Description |
|------|-------------|
| `SSE-C` | Encryption keys managed by Plik |
| `S3` | Encryption keys managed by the S3 backend |

```toml
[DataBackendConfig]
    SSE = "SSE-C"  # or "S3"
```

## OpenStack Swift

```toml
DataBackend = "swift"
[DataBackendConfig]
    Container = "plik"
    AuthUrl = "https://auth.swiftapi.example.com/v2.0/"
    UserName = "user@example.com"
    ApiKey = "xxxxxxxxxxxxxxxx"
    Domain = "domain"   # v3 auth only
    Tenant = "tenant"   # v2 auth only
```

See the [ncw/swift documentation](https://github.com/ncw/swift) for all available connection settings (v1/v2/v3).

## Google Cloud Storage

```toml
DataBackend = "gcs"
[DataBackendConfig]
    Bucket = "my-plik-bucket"
    Folder = "plik"
```

Requires Application Default Credentials or a service account key.
