# Qdrant Controller

A Kubernetes controller for managing Qdrant Cloud clusters declaratively.

## Status

✅ **Implementation Complete!** The controller is fully functional and ready for testing.

## Features

- 🎯 **Declarative Cluster Management** - Define Qdrant clusters as Kubernetes resources
- 🔐 **Multi-Tenancy** - Per-resource credentials (no global API keys)
- 📦 **Hybrid Package Selection** - Explicit packageID or auto-select from resource requirements
- 🔑 **Auto Database Key Creation** - Automatically generates database API keys
- 🔌 **Connection Secrets** - Auto-creates secrets with REST and gRPC endpoints
- ⏸️ **Suspend/Resume** - Full cluster lifecycle management
- 📊 **Status Tracking** - Comprehensive phase tracking and conditions
- 🧹 **Finalizers** - Proper cleanup on deletion

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.27+)
- Qdrant Cloud account and Management API key
- `kubectl` configured to access your cluster

### Installation

#### Using Helm (Recommended)

1. **Install CRDs and controller:**
   ```bash
   helm upgrade --install qdrant-controller oci://ghcr.io/doodlescheduling/charts/qdrant-controller \
     -n qdrant-controller-system --create-namespace
   ```

2. **Create a secret with your Qdrant Cloud Management API key:**
   ```bash
   kubectl create secret generic qdrant-cloud-api-key \
     -n default \
     --from-literal=apiKey='your-management-api-key-here'
   ```

#### Manual Installation

1. **Install CRDs:**
   ```bash
   make install
   ```

2. **Create a secret with your Qdrant Cloud Management API key:**
   ```bash
   kubectl create secret generic qdrant-cloud-api-key \
     --from-literal=apiKey='your-management-api-key-here'
   ```

3. **Deploy the controller:**
   ```bash
   make deploy IMG=your-registry/qdrant-controller:latest
   ```

   Or run locally for development:
   ```bash
   make run
   ```

### Create Your First Cluster

**Option 1: Auto-select package from resource requirements**
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
      ram: "8GiB"
      cpu: "2000m"
      disk: "32GiB"
  storageTier: balanced
  secret:
    name: qdrant-cloud-api-key
```

**Option 2: Explicit package ID**
```yaml
apiVersion: qdrant.infra.doodle.com/v1beta1
kind: QdrantCluster
metadata:
  name: my-cluster
spec:
  accountID: "your-account-uuid"
  cloudProvider: aws
  cloudRegion: us-east-1
  nodeCount: 1
  packageSelection:
    packageID: "package-uuid-from-qdrant-cloud"
  secret:
    name: qdrant-cloud-api-key
```

**Apply the manifest:**
```bash
kubectl apply -f config/samples/qdrant_v1beta1_qdrantcluster_resource_based.yaml
```

**Check status:**
```bash
kubectl get qdrantclusters
kubectl describe qdrantcluster my-cluster
```

**Get connection details:**
```bash
kubectl get secret my-cluster-connection -o yaml
```

## Development

### Build
```bash
make build
```

### Run Tests
```bash
make test
```

### Generate Code
```bash
make generate  # Generate deepcopy code
make manifests # Generate CRDs and RBAC
```

### Run Locally
```bash
make run
```

## Documentation

- [Helm Chart README](chart/qdrant-controller/README.md) - Helm chart documentation
- [AGENT_SPEC.md](docs/AGENT_SPEC.md) - Complete implementation specification
- [API_RESEARCH_FINDINGS.md](docs/API_RESEARCH_FINDINGS.md) - Qdrant Cloud API research
- [MULTI_TENANCY.md](docs/MULTI_TENANCY.md) - Multi-tenancy pattern documentation
- [Examples](config/samples/) - Example manifests
- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) - Community guidelines
- [SECURITY.md](SECURITY.md) - Security policy

## Architecture

The controller follows Kubernetes operator patterns using [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime):

- **CRD**: `QdrantCluster` defines the desired state
- **Controller**: Reconciles actual state with desired state
- **gRPC Client**: Communicates with Qdrant Cloud API
- **Package Selector**: Auto-selects packages based on resource requirements

## References

- [Qdrant Cloud Documentation](https://qdrant.tech/documentation/cloud-quickstart/)
- [Qdrant Cloud Public API Repository](https://github.com/qdrant/qdrant-cloud-public-api)

## License

Apache License 2.0
