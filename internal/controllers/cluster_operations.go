package controllers

import (
	"context"
	"fmt"
	"time"

	infrav1beta1 "github.com/doodlescheduling/qdrant-controller/api/v1beta1"
	qdrantclient "github.com/doodlescheduling/qdrant-controller/pkg/qdrant/client"
	"github.com/go-logr/logr"
	authv2 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/cluster/auth/v2"
	clusterv1 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/cluster/v1"
	commonv1 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *QdrantClusterReconciler) createCluster(ctx context.Context, qdrant *qdrantclient.Client, cluster *infrav1beta1.QdrantCluster, logger logr.Logger) error {
	// Resolve package ID if using resource requirements
	packageID, err := r.resolvePackageID(ctx, qdrant, cluster, logger)
	if err != nil {
		return fmt.Errorf("failed to resolve package ID: %w", err)
	}

	// Build cluster configuration
	config := &clusterv1.ClusterConfiguration{
		NumberOfNodes: uint32(cluster.Spec.NodeCount),
		PackageId:     packageID,
	}

	// Add version if specified
	if cluster.Spec.QdrantVersion != "" {
		config.Version = &cluster.Spec.QdrantVersion
	}

	// Add additional disk if specified
	if cluster.Spec.AdditionalDiskGiB != nil && *cluster.Spec.AdditionalDiskGiB > 0 {
		config.AdditionalResources = &clusterv1.AdditionalResources{
			Disk: uint32(*cluster.Spec.AdditionalDiskGiB),
		}
	}

	// Add storage tier if specified
	if cluster.Spec.StorageTier != "" {
		config.ClusterStorageConfiguration = &clusterv1.ClusterStorageConfiguration{
			StorageTierType: qdrantclient.ConvertStorageTier(string(cluster.Spec.StorageTier)),
		}
	}

	// Note: DatabaseConfiguration from spec is skipped for now as it requires complex structured config
	// This can be added in a future enhancement

	// Build labels
	var labels []*commonv1.KeyValue
	if cluster.Spec.Configuration != nil && cluster.Spec.Configuration.Labels != nil {
		for key, value := range cluster.Spec.Configuration.Labels {
			labels = append(labels, &commonv1.KeyValue{
				Key:   key,
				Value: value,
			})
		}
	}

	// Build the Cluster object
	qdrantCluster := &clusterv1.Cluster{
		AccountId:             cluster.Spec.AccountID,
		Name:                  cluster.Name,
		CloudProviderId:       string(cluster.Spec.CloudProvider),
		CloudProviderRegionId: cluster.Spec.CloudRegion,
		Configuration:         config,
		Labels:                labels,
	}

	// Create cluster request
	req := &clusterv1.CreateClusterRequest{
		Cluster: qdrantCluster,
	}

	resp, err := qdrant.CreateCluster(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	if resp.Cluster == nil {
		return fmt.Errorf("create cluster response missing cluster data")
	}

	// Update status with cluster ID
	cluster.Status.ClusterID = resp.Cluster.Id
	cluster.Status.PackageID = packageID

	// Update status from cluster state
	*cluster = r.updateStatusFromCluster(*cluster, resp.Cluster)

	logger.Info("cluster created", "clusterID", cluster.Status.ClusterID)
	return nil
}

func (r *QdrantClusterReconciler) updateCluster(ctx context.Context, qdrant *qdrantclient.Client, cluster *infrav1beta1.QdrantCluster, logger logr.Logger) error {
	// Resolve package ID if using resource requirements
	packageID, err := r.resolvePackageID(ctx, qdrant, cluster, logger)
	if err != nil {
		return fmt.Errorf("failed to resolve package ID: %w", err)
	}

	// Build cluster configuration
	config := &clusterv1.ClusterConfiguration{
		NumberOfNodes: uint32(cluster.Spec.NodeCount),
		PackageId:     packageID,
	}

	// Add version if specified
	if cluster.Spec.QdrantVersion != "" {
		config.Version = &cluster.Spec.QdrantVersion
	}

	// Add additional disk if specified
	if cluster.Spec.AdditionalDiskGiB != nil && *cluster.Spec.AdditionalDiskGiB > 0 {
		config.AdditionalResources = &clusterv1.AdditionalResources{
			Disk: uint32(*cluster.Spec.AdditionalDiskGiB),
		}
	}

	// Add storage tier if specified
	if cluster.Spec.StorageTier != "" {
		config.ClusterStorageConfiguration = &clusterv1.ClusterStorageConfiguration{
			StorageTierType: qdrantclient.ConvertStorageTier(string(cluster.Spec.StorageTier)),
		}
	}

	// Note: DatabaseConfiguration from spec is skipped for now as it requires complex structured config
	// This can be added in a future enhancement

	// Build labels
	var labels []*commonv1.KeyValue
	if cluster.Spec.Configuration != nil && cluster.Spec.Configuration.Labels != nil {
		for key, value := range cluster.Spec.Configuration.Labels {
			labels = append(labels, &commonv1.KeyValue{
				Key:   key,
				Value: value,
			})
		}
	}

	// Build the Cluster object
	qdrantCluster := &clusterv1.Cluster{
		Id:                    cluster.Status.ClusterID,
		AccountId:             cluster.Spec.AccountID,
		Name:                  cluster.Name,
		CloudProviderId:       string(cluster.Spec.CloudProvider),
		CloudProviderRegionId: cluster.Spec.CloudRegion,
		Configuration:         config,
		Labels:                labels,
	}

	// Update cluster request
	req := &clusterv1.UpdateClusterRequest{
		Cluster: qdrantCluster,
	}

	resp, err := qdrant.UpdateCluster(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	if resp.Cluster == nil {
		return fmt.Errorf("update cluster response missing cluster data")
	}

	// Update status
	cluster.Status.PackageID = packageID
	*cluster = r.updateStatusFromCluster(*cluster, resp.Cluster)

	logger.Info("cluster updated", "clusterID", cluster.Status.ClusterID)
	return nil
}

func (r *QdrantClusterReconciler) resolvePackageID(ctx context.Context, qdrant *qdrantclient.Client, cluster *infrav1beta1.QdrantCluster, logger logr.Logger) (string, error) {
	// If packageID is explicitly specified, use it
	if cluster.Spec.PackageSelection.PackageID != nil && *cluster.Spec.PackageSelection.PackageID != "" {
		return *cluster.Spec.PackageSelection.PackageID, nil
	}

	// If resource requirements are specified, find matching package
	if cluster.Spec.PackageSelection.ResourceRequirements != nil {
		logger.Info("resolving package from resource requirements")

		// List available packages
		packagesResp, err := qdrant.ListPackages(ctx, cluster.Spec.AccountID, string(cluster.Spec.CloudProvider), cluster.Spec.CloudRegion)
		if err != nil {
			return "", fmt.Errorf("failed to list packages: %w", err)
		}

		// Select package based on requirements
		selector := qdrantclient.NewPackageSelector(packagesResp.Items)
		reqs := cluster.Spec.PackageSelection.ResourceRequirements
		pkg, err := selector.SelectPackage(reqs.RAM, reqs.CPU, reqs.Disk)
		if err != nil {
			return "", fmt.Errorf("failed to select package: %w", err)
		}

		logger.Info("selected package", "packageID", pkg.Id, "name", pkg.Name, "ram", pkg.ResourceConfiguration.Ram, "cpu", pkg.ResourceConfiguration.Cpu, "disk", pkg.ResourceConfiguration.Disk)
		return pkg.Id, nil
	}

	return "", fmt.Errorf("either packageID or resourceRequirements must be specified")
}

func (r *QdrantClusterReconciler) createDatabaseKey(ctx context.Context, qdrant *qdrantclient.Client, cluster *infrav1beta1.QdrantCluster, logger logr.Logger) error {
	keyName := fmt.Sprintf("%s-key", cluster.Name)
	if cluster.Spec.DatabaseKeyConfig != nil && cluster.Spec.DatabaseKeyConfig.Name != "" {
		keyName = cluster.Spec.DatabaseKeyConfig.Name
	}

	// Note: Scopes from spec are currently not used as the API uses AccessRules instead
	// This can be enhanced in the future to convert scopes to AccessRules

	expiresInDays := int32(90)
	if cluster.Spec.DatabaseKeyConfig != nil && cluster.Spec.DatabaseKeyConfig.ExpiresInDays != nil {
		expiresInDays = *cluster.Spec.DatabaseKeyConfig.ExpiresInDays
	}

	expiresAt := time.Now().AddDate(0, 0, int(expiresInDays))

	// Build the DatabaseApiKey object
	dbKey := &authv2.DatabaseApiKey{
		AccountId: cluster.Spec.AccountID,
		ClusterId: cluster.Status.ClusterID,
		Name:      keyName,
		ExpiresAt: timestamppb.New(expiresAt),
		// AccessRules would go here for fine-grained permissions
	}

	req := &authv2.CreateDatabaseApiKeyRequest{
		DatabaseApiKey: dbKey,
	}

	resp, err := qdrant.CreateDatabaseApiKey(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create database API key: %w", err)
	}

	if resp.DatabaseApiKey == nil {
		return fmt.Errorf("create database API key response missing key data")
	}

	cluster.Status.DatabaseKeyID = resp.DatabaseApiKey.Id
	logger.Info("database API key created", "keyID", resp.DatabaseApiKey.Id)

	return nil
}

func (r *QdrantClusterReconciler) ensureConnectionSecret(ctx context.Context, cluster *infrav1beta1.QdrantCluster, secretName string, qdrantCluster *clusterv1.Cluster) error {
	if qdrantCluster.State == nil || qdrantCluster.State.Endpoint == nil {
		return fmt.Errorf("cluster endpoint not available yet")
	}

	endpoint := qdrantCluster.State.Endpoint

	// Get database API key from existing secret if it exists
	var existingSecret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: cluster.Namespace,
	}, &existingSecret)

	var apiKey string
	if err == nil {
		// Secret exists, preserve API key
		apiKey = string(existingSecret.Data["apiKey"])
	} else if !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to get existing secret: %w", err)
	}

	// Build connection URLs
	restEndpoint := fmt.Sprintf("%s:%d", endpoint.Url, endpoint.RestPort)
	grpcEndpoint := fmt.Sprintf("%s:%d", endpoint.Url, endpoint.GrpcPort)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: cluster.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"endpoint":     []byte(restEndpoint),
			"grpcEndpoint": []byte(grpcEndpoint),
		},
	}

	if apiKey != "" {
		secret.Data["apiKey"] = []byte(apiKey)
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(cluster, secret, r.Scheme()); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create or update secret
	if err == nil {
		// Update existing secret
		if err := r.Update(ctx, secret); err != nil {
			return fmt.Errorf("failed to update secret: %w", err)
		}
	} else {
		// Create new secret
		if err := r.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
	}

	cluster.Status.ConnectionSecret = secretName
	return nil
}

func (r *QdrantClusterReconciler) updateStatusFromCluster(cluster infrav1beta1.QdrantCluster, qdrantCluster *clusterv1.Cluster) infrav1beta1.QdrantCluster {
	if qdrantCluster.State != nil {
		cluster.Status.Phase = qdrantclient.ConvertPhaseToString(qdrantCluster.State.Phase)
		cluster.Status.Version = qdrantCluster.State.Version
		cluster.Status.NodesUp = int32(qdrantCluster.State.NodesUp)

		if qdrantCluster.State.Endpoint != nil {
			cluster.Status.Endpoint = &infrav1beta1.ClusterEndpoint{
				URL:      qdrantCluster.State.Endpoint.Url,
				RESTPort: qdrantCluster.State.Endpoint.RestPort,
				GRPCPort: qdrantCluster.State.Endpoint.GrpcPort,
			}
		}
	}

	return cluster
}

func (r *QdrantClusterReconciler) needsUpdate(cluster infrav1beta1.QdrantCluster, qdrantCluster *clusterv1.Cluster) bool {
	if qdrantCluster.Configuration == nil {
		return false
	}

	// Check node count
	if int32(qdrantCluster.Configuration.NumberOfNodes) != cluster.Spec.NodeCount {
		return true
	}

	// Check package ID if it's resolved
	if cluster.Status.PackageID != "" && qdrantCluster.Configuration.PackageId != cluster.Status.PackageID {
		return true
	}

	// Check version if specified
	if cluster.Spec.QdrantVersion != "" && qdrantCluster.Configuration.Version != nil {
		if *qdrantCluster.Configuration.Version != cluster.Spec.QdrantVersion {
			return true
		}
	}

	// Check additional disk
	if cluster.Spec.AdditionalDiskGiB != nil {
		if qdrantCluster.Configuration.AdditionalResources == nil {
			return *cluster.Spec.AdditionalDiskGiB != 0
		}
		if int32(qdrantCluster.Configuration.AdditionalResources.Disk) != *cluster.Spec.AdditionalDiskGiB {
			return true
		}
	}

	// Check storage tier
	if cluster.Spec.StorageTier != "" {
		if qdrantCluster.Configuration.ClusterStorageConfiguration == nil {
			return true
		}
		expectedTier := qdrantclient.ConvertStorageTier(string(cluster.Spec.StorageTier))
		if qdrantCluster.Configuration.ClusterStorageConfiguration.StorageTierType != expectedTier {
			return true
		}
	}

	return false
}
