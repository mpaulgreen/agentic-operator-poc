# Catalog Management

Catalogs are collections of operator bundles that OLM uses to discover and install operators.

## Catalog Types

### Index Image (OLM v0)
A container image containing a SQLite database of operator bundles.

```bash
# Build index image with a bundle
opm index add \
  --bundles quay.io/example/redis-operator-bundle:v0.1.0 \
  --tag quay.io/example/redis-operator-index:latest

# Add new version to existing index
opm index add \
  --bundles quay.io/example/redis-operator-bundle:v0.2.0 \
  --from-index quay.io/example/redis-operator-index:latest \
  --tag quay.io/example/redis-operator-index:latest
```

### File-Based Catalog (OLM v1)
A declarative YAML/JSON catalog format.

```bash
# Initialize catalog
opm init redis-operator --default-channel=alpha --output yaml > catalog.yaml

# Render bundle into catalog
opm render quay.io/example/redis-operator-bundle:v0.1.0 --output yaml >> catalog.yaml

# Validate
opm validate catalog/
```

## CatalogSource

Deploy a catalog to the cluster:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: redis-operator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/example/redis-operator-index:latest
  displayName: Redis Operator
  publisher: Example Inc
  updateStrategy:
    registryPoll:
      interval: 10m
```

## Channels

Channels control which versions users can subscribe to:

| Channel | Purpose | Example Versions |
|---------|---------|-----------------|
| alpha | Early testing, breaking changes allowed | 0.1.0, 0.2.0, 0.3.0 |
| beta | Feature complete, API stable | 0.5.0, 0.6.0 |
| stable | Production ready, backwards compatible | 1.0.0, 1.1.0, 1.2.0 |

An operator can be in multiple channels simultaneously. The `annotations.yaml` lists channels:
```yaml
operators.operatorframework.io.bundle.channels.v1: alpha,beta
operators.operatorframework.io.bundle.channel.default.v1: alpha
```

## Subscription

Users install operators by creating a Subscription:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: redis-operator
  namespace: operators
spec:
  channel: alpha
  name: redis-operator
  source: redis-operator-catalog
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic   # or Manual
```

## Upgrade Graph

OLM builds an upgrade graph from CSV `replaces` fields:

```
v0.1.0 ← v0.2.0 ← v0.3.0 ← v1.0.0
```

Each CSV points to the one it replaces. `skips` allows skipping intermediate versions:

```yaml
spec:
  version: 1.0.0
  replaces: redis-operator.v0.3.0
  skips:
  - redis-operator.v0.2.0    # Users on 0.2.0 jump directly to 1.0.0
```
