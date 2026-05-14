# elasticsearch-operator

A Kubernetes operator for managing ElasticsearchCluster resources.

## Description

This operator manages the lifecycle of ElasticsearchCluster custom resources on Kubernetes and OpenShift clusters.

## Getting Started

### Prerequisites
- go version v1.22+
- docker/podman
- kubectl v1.29+
- Access to a Kubernetes v1.29+ cluster

### Running on the cluster

1. Install the CRDs:
```sh
make install
```

2. Build and push the image:
```sh
make docker-build docker-push IMG=<registry>/elasticsearch-operator:tag
```

3. Deploy the operator:
```sh
make deploy IMG=<registry>/elasticsearch-operator:tag
```

### Uninstall

```sh
make undeploy
```

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0.
