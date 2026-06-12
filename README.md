# poweradmin-operator

A Kubernetes operator for managing DNS zones and records via
[Poweradmin](https://www.poweradmin.org/). Built on top of
[poweradmin-go](https://github.com/Contentways/poweradmin-go).

## Features

- **`DNSZone`** — create, update, and delete DNS zones in Poweradmin
- **`DNSRecord`** — manage individual DNS records (A, AAAA, CNAME, MX, TXT, ...)
- **Per-namespace credentials** — each namespace uses its own Poweradmin account via a Kubernetes Secret, making the operator safe for multi-tenant clusters
- **Finalizer-based cleanup** — zones and records are deleted from Poweradmin before the Kubernetes resource is removed

## Installation

### Prerequisites

- Kubernetes 1.28+
- Helm 3.x
- A running Poweradmin 4.3.0+ instance with API v2

### Install via Helm

```sh
helm install poweradmin-operator oci://registry-1.docker.io/contentwaysorg/poweradmin-operator-chart \
  --namespace poweradmin-operator \
  --create-namespace
```

### Create credentials per namespace

The operator reads Poweradmin credentials from a Secret named
`poweradmin-credentials` (configurable) in the same namespace as the CR.

```sh
kubectl create secret generic poweradmin-credentials \
  --from-literal=POWERADMIN_URL=https://dns.example.com \
  --from-literal=POWERADMIN_API_KEY=your-api-key \
  --namespace default
```

## Quickstart

### Create a DNS zone

```yaml
apiVersion: dns.contentways.org/v1alpha1
kind: DNSZone
metadata:
  name: example-org
  namespace: default
spec:
  name: example.org
  type: NATIVE
  nameservers:
    - ns1.example.org
    - ns2.example.org
```

```sh
kubectl apply -f dnszone.yaml
kubectl get dnszones
```

```
NAME          ZONE         TYPE     ID    READY
example-org   example.org  NATIVE   42    True
```

### Create a DNS record

```yaml
apiVersion: dns.contentways.org/v1alpha1
kind: DNSRecord
metadata:
  name: www-example-org
  namespace: default
spec:
  zoneName: example.org
  name: www
  type: A
  content: 1.2.3.4
  ttl: 3600
```

```sh
kubectl apply -f dnsrecord.yaml
kubectl get dnsrecords
```

```
NAME              ZONE         NAME   TYPE   CONTENT   TTL    READY
www-example-org   example.org  www    A      1.2.3.4   3600   True
```

## API Reference

### DNSZone

| Field                | Type     | Required | Description                                                   |
| -------------------- | -------- | -------- | ------------------------------------------------------------- |
| `spec.name`          | string   | ✓        | DNS zone name (e.g. `example.org`)                            |
| `spec.type`          | string   |          | Zone type: `NATIVE`, `MASTER`, `SLAVE` (default: `NATIVE`)    |
| `spec.nameservers`   | []string | ✓        | Nameservers for the zone (e.g. `["ns1.example.org"]`)         |
| `spec.masters`       | string   |          | Master server IPs for SLAVE zones                             |

| Status field               | Description                              |
| -------------------------- | ---------------------------------------- |
| `status.zoneId`            | Numeric ID assigned by Poweradmin        |
| `status.conditions[Ready]` | `True` when the zone is in sync          |

### DNSRecord

| Field              | Type    | Required | Description                                                                          |
| ------------------ | ------- | -------- | ------------------------------------------------------------------------------------ |
| `spec.zoneName`    | string  | ✓        | Name of the parent DNS zone                                                          |
| `spec.name`        | string  | ✓        | Record name (e.g. `www`)                                                             |
| `spec.type`        | string  | ✓        | Record type: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `SRV`, `CAA`, `PTR`, `SOA`   |
| `spec.content`     | string  | ✓        | Record value                                                                         |
| `spec.ttl`         | integer |          | TTL in seconds (default: `3600`)                                                     |
| `spec.priority`    | integer |          | Priority for MX/SRV records (default: `0`)                                           |
| `spec.disabled`    | boolean |          | Disable the record (default: `false`)                                                |

| Status field               | Description                              |
| -------------------------- | ---------------------------------------- |
| `status.recordId`          | Numeric ID assigned by Poweradmin        |
| `status.zoneId`            | Numeric ID of the parent zone            |
| `status.conditions[Ready]` | `True` when the record is in sync        |

## Helm Values

| Value                              | Default                                       | Description                          |
| ---------------------------------- | --------------------------------------------- | ------------------------------------ |
| `image.repository`                 | `contentwaysorg/poweradmin-operator`          | Container image                      |
| `image.tag`                        | Chart appVersion                              | Image tag                            |
| `poweradmin.credentialsSecretName` | `poweradmin-credentials`                      | Secret name to look up per namespace |
| `replicaCount`                     | `1`                                            | Number of operator replicas          |
| `leaderElection.enabled`           | `false`                                        | Enable leader election for HA        |
| `crds.install`                     | `true`                                         | Install CRDs as Helm hooks           |
| `metrics.enabled`                  | `true`                                         | Expose Prometheus metrics            |

## Multi-tenancy

Each namespace manages its own Poweradmin credentials — a DNSZone or DNSRecord
in namespace `team-a` will use the Secret `poweradmin-credentials` from
`team-a`, completely isolated from other namespaces.

```
namespace: team-a          namespace: team-b
  Secret → account-a         Secret → account-b
  DNSZone → zone-a.example   DNSZone → zone-b.example
```

## Development

```sh
# Run tests
make test

# Run locally against a kind cluster
kind create cluster --name poweradmin-operator
make install
POWERADMIN_URL=https://dns.example.com \
POWERADMIN_API_KEY=your-key \
make run
```

Pre-commit hooks are configured in
[`.pre-commit-config.yaml`](.pre-commit-config.yaml):

```sh
pip install pre-commit
pre-commit install
```

## Related

- [poweradmin-go](https://github.com/Contentways/poweradmin-go) — Go SDK used by this operator
- [Poweradmin](https://www.poweradmin.org/) — web-based DNS administration frontend for PowerDNS

## License

[Apache 2.0](LICENSE) © 2026 Patrick Omland
