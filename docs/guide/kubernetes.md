# Kubernetes (Helm)

> [!IMPORTANT]
> **Data Safety Warning**: For production deployments, we strongly advise using a **dedicated SQL database** (PostgreSQL or MySQL) for metadata, and enabling **persistence** (PVC) or using a **remote backend** (S3, GCS, Swift) for file storage to ensure no data is lost.

Plik provides a Helm chart to deploy it on Kubernetes.

The chart source is located in `charts/plik/`.

## Installation

### From Repository

```bash
helm repo add plik https://root-gg.github.io/plik
helm repo update
helm install plik plik/plik
```

### From Source
```bash
git clone https://github.com/root-gg/plik.git
cd plik/charts/plik
helm install plik .
```

## Configuration

The chart supports several modes of deployment. All Plik [configuration](./configuration.md) options are available under the `plikd` key in `values.yaml`.

### Values Reference

A complete table of all configurable values with types, defaults, and descriptions is available in the chart's [README](https://github.com/root-gg/plik/blob/master/charts/plik/README.md#values).

The README is auto-generated with [helm-docs](https://github.com/norwoodj/helm-docs). To regenerate after editing `values.yaml`:

```bash
make helm-docs
```

### Persistence

Plik requires two types of persistent storage when using the `file` data backend:

| PVC | `values.yaml` key | Default mount path | Stores |
|-----|------------------|--------------------|--------|
| File data | `persistence` | `/home/plik/server/files` | Uploaded files |
| Database | `dbPersistence` | `/home/plik/server/db` | SQLite metadata (`plik.db`) |

By default both use `emptyDir` (data is lost on pod restart). For production, enable both.

> [!WARNING]
> If you only enable `persistence` (file data PVC) but not `dbPersistence`, the SQLite database will still be on `emptyDir`. Restarting the pod will delete all metadata — uploaded files will survive on the PV but Plik will not know about them.

#### StatefulSet (Recommended when using File Data Backend)

`StatefulSet` is recommended for the `file` backend as each pod gets its own stable PVCs via `volumeClaimTemplates`.

```yaml
kind: StatefulSet

persistence:
  size: 10Gi

dbPersistence:
  size: 1Gi
```

#### Deployment (Recommended when using Cloud Data Backends)

If you use S3, GCS, or Swift for file storage, a standard `Deployment` is more appropriate. Enable `dbPersistence` to preserve the SQLite database across pod restarts.

```yaml
kind: Deployment

dbPersistence:
  enabled: true
  size: 1Gi

plikd:
  DataBackend: s3
  DataBackendConfig:
    Endpoint: "s3.amazonaws.com"
    Bucket: "my-bucket"
    # Credentials should be provided via secret
```

#### Custom Database Connection String

The default `MetadataBackendConfig.ConnectionString` is `/home/plik/server/db/plik.db` (inside the `dbPersistence` volume). Override it to use PostgreSQL or MySQL instead:

```yaml
plikd:
  MetadataBackendConfig:
    Driver: "postgres"
    ConnectionString: "host=pg.example.com user=plik dbname=plik sslmode=require"
```

When using a remote database, `dbPersistence` is not needed.

### Secrets Management

Sensitive values (API secrets, database passwords) are managed via a Kubernetes Secret. You can either provide them in `values.yaml` (which will be automatically extracted into a Secret and injected as environment variables) or specify an `existingSecret`.

```yaml
# Use an existing secret containing PLIKD_GOOGLE_API_SECRET, etc.
existingSecret: "my-custom-plik-secret"
```

The following environment variables are supported for secret injection:

| Environment Variable | Values Key |
|---------------------|------------|
| `PLIKD_GOOGLE_API_SECRET` | `plikd.GoogleApiSecret` |
| `PLIKD_OVH_API_SECRET` | `plikd.OvhApiSecret` |
| `PLIKD_OVH_API_KEY` | `plikd.OvhApiKey` |
| `PLIKD_OIDC_CLIENT_SECRET` | `plikd.OIDCClientSecret` |
| `PLIKD_DATA_BACKEND_CONFIG` | `plikd.DataBackendConfig` (sensitive keys as JSON) |
| `PLIKD_METADATA_BACKEND_CONFIG` | `plikd.MetadataBackendConfig` (sensitive keys as JSON) |

## Service and Ingress

The service is exposed via a `ClusterIP` by default. You can configure an `Ingress` in `values.yaml`.

```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: plik.example.com
      paths:
        - path: /
          pathType: ImplementationSpecific
  tls:
    - secretName: plik-tls
      hosts:
        - plik.example.com
```

## Upgrading

When upgrading the chart, use:

```bash
helm repo update
helm upgrade plik plik/plik
```
