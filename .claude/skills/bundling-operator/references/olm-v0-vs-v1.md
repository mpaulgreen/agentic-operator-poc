# OLM v0 vs v1

## Current: OLM v0 (operator-framework v0.x)

This is the stable, widely deployed version used in OpenShift 4.x.

### Key Concepts
- **CatalogSource**: Points to an index image containing operator bundles
- **Subscription**: Declares intent to install an operator from a catalog
- **InstallPlan**: Created by OLM to track installation steps
- **CSV**: The primary artifact describing the operator
- **Bundle format**: `registry+v1` — directories with manifests/ and metadata/

### Upgrade Model
- `spec.replaces`: Points to the previous version (creates a linked list)
- `spec.skips`: Skip intermediate versions in the upgrade path
- Channels (alpha, beta, stable) control which versions are available
- OLM resolves the upgrade graph and creates an InstallPlan

### Index Images
- Built with `opm index add --bundles <bundle-image>`
- Contains a SQLite database of all operator bundles
- Referenced by CatalogSource in the cluster

## Emerging: OLM v1 (operator-controller)

OLM v1 is under active development and changes some fundamental patterns.

### Key Differences from v0
- **ClusterExtension** replaces Subscription as the install intent API
- **File-Based Catalogs (FBC)** replace SQLite index images
- **Declarative config** replaces imperative `opm` commands
- **Semver-based upgrades** replace the replaces/skips linked list
- **No more InstallPlan** — upgrades happen automatically based on constraints

### Migration Path
- Bundle format remains compatible (`registry+v1`)
- CSV structure unchanged — same skill output works for both
- Catalog creation changes (FBC YAML instead of SQLite)
- Install API changes (ClusterExtension instead of Subscription)

### When to Use Which
- **v0**: OpenShift 4.x clusters, current production
- **v1**: Future clusters, tech preview on some platforms
- **This skill**: Generates bundles compatible with both (the bundle format is shared)

## Recommendation

Generate bundles using the v0 format (`registry+v1`). This is forward-compatible with OLM v1. The only difference is how the bundle is added to a catalog — FBC vs index image — which is a deployment concern, not a bundle concern.
