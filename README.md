# qdrant-controller

[![release](https://img.shields.io/github/release/DoodleScheduling/qdrant-controller/all.svg)](https://github.com/DoodleScheduling/qdrant-controller/releases)
[![release](https://github.com/DoodleScheduling/qdrant-controller/actions/workflows/release.yaml/badge.svg)](https://github.com/DoodleScheduling/qdrant-controller/actions/workflows/release.yaml)
[![report](https://goreportcard.com/badge/github.com/DoodleScheduling/qdrant-controller)](https://goreportcard.com/report/github.com/DoodleScheduling/qdrant-controller)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/qdrant-controller/badge)](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/qdrant-controller)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/qdrant-controller/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/qdrant-controller?branch=master)
[![license](https://img.shields.io/github/license/DoodleScheduling/qdrant-controller.svg)](https://github.com/DoodleScheduling/qdrant-controller/blob/master/LICENSE)

Kubernetes controller for managing Qdrant Cloud clusters.

## Quickstart

### Usage Example

```yaml
apiVersion: qdrant.infra.doodle.com/v1beta1
kind: QdrantCluster
metadata:
  name: my-cluster
spec:
  accountID: "your-account-uuid"
  cloudProvider: aws
  cloudRegion: us-east-1
  nodeCount: 3
  packageSelection:
    resourceRequirements:
      ram: "8Gi"
      cpu: "2"
      disk: "32Gi"
  secret:
    name: qdrant-cloud-api-key
---
apiVersion: v1
kind: Secret
metadata:
  name: qdrant-cloud-api-key
data:
  apiKey: <base64-encoded-management-api-key>
type: Opaque
```

Alternatively, specify a package UUID directly instead of resource requirements:

```yaml
spec:
  packageSelection:
    packageID: "package-uuid-from-qdrant-cloud"
```

## Observe reconciliation

Each resource reports various conditions in `.status.conditions` which will give the necessary insight about the
current state of the resource.

```yaml
status:
  conditions:
  - lastTransitionTime: "2024-01-15T10:30:00Z"
    message: ""
    observedGeneration: 1
    reason: ReconciliationSucceeded
    status: "True"
    type: Ready
```

## Installation

### Helm

Please see [chart/qdrant-controller](https://github.com/DoodleScheduling/qdrant-controller/tree/master/chart/qdrant-controller) for the helm chart docs.

### Manifests/kustomize

Alternatively you may get the bundled manifests in each release to deploy it using kustomize or use them directly.

## Configuration
The controller can be configured using cmd args:
```
      --concurrent int                            The number of concurrent reconciles. (default 4)
      --enable-leader-election                    Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
      --graceful-shutdown-timeout duration        The duration given to the reconciler to finish before forcibly stopping. (default 10m0s)
      --health-addr string                        The address the health endpoint binds to. (default ":9557")
      --insecure-kubeconfig-exec                  Allow use of the user.exec section in kubeconfigs provided for remote apply.
      --insecure-kubeconfig-tls                   Allow that kubeconfigs provided for remote apply can disable TLS verification.
      --kube-api-burst int                        The maximum burst queries-per-second of requests sent to the Kubernetes API. (default 300)
      --kube-api-qps float32                      The maximum queries-per-second of requests sent to the Kubernetes API. (default 50)
      --leader-election-lease-duration duration   Interval at which non-leader candidates will wait to force acquire leadership (duration string). (default 35s)
      --leader-election-release-on-cancel         Defines if the leader should step down voluntarily on controller manager shutdown. (default true)
      --leader-election-renew-deadline duration   Duration that the leading controller manager will retry refreshing leadership before giving up (duration string). (default 30s)
      --leader-election-retry-period duration     Duration the LeaderElector clients should wait between tries of actions (duration string). (default 5s)
      --log-encoding string                       Log encoding format. Can be 'json' or 'console'. (default "json")
      --log-level string                          Log verbosity level. Can be one of 'trace', 'debug', 'info', 'error'. (default "info")
      --max-retry-delay duration                  The maximum amount of time for which an object being reconciled will have to wait before a retry. (default 15m0s)
      --metrics-addr string                       The address the metric endpoint binds to. (default ":9556")
      --min-retry-delay duration                  The minimum amount of time for which an object being reconciled will have to wait before a retry. (default 750ms)
      --watch-all-namespaces                      Watch for resources in all namespaces, if set to false it will only watch the runtime namespace. (default true)
      --watch-label-selector string               Watch for resources with matching labels e.g. 'sharding.fluxcd.io/shard=shard1'.
```
