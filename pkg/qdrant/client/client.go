package client

import (
	"context"
	"fmt"

	bookingv1 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/booking/v1"
	authv2 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/cluster/auth/v2"
	clusterv1 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/cluster/v1"
	commonv1 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	// DefaultEndpoint is the default Qdrant Cloud API endpoint
	DefaultEndpoint = "cloud.qdrant.io:443"
)

// Client wraps the Qdrant Cloud gRPC clients with authentication
type Client struct {
	apiKey string
	conn   *grpc.ClientConn

	// Service clients
	ClusterService     clusterv1.ClusterServiceClient
	BookingService     bookingv1.BookingServiceClient
	DatabaseKeyService authv2.DatabaseApiKeyServiceClient
}

// NewClient creates a new Qdrant Cloud API client
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	config := &Config{
		Endpoint: DefaultEndpoint,
	}

	for _, opt := range opts {
		opt(config)
	}

	// Create gRPC connection with TLS
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}

	conn, err := grpc.NewClient(config.Endpoint, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	client := &Client{
		apiKey:             apiKey,
		conn:               conn,
		ClusterService:     clusterv1.NewClusterServiceClient(conn),
		BookingService:     bookingv1.NewBookingServiceClient(conn),
		DatabaseKeyService: authv2.NewDatabaseApiKeyServiceClient(conn),
	}

	return client, nil
}

// Close closes the underlying gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// withAuth adds the API key to the context
func (c *Client) withAuth(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "apikey "+c.apiKey)
}

// ListClusters lists all clusters for an account
func (c *Client) ListClusters(ctx context.Context, accountID string) (*clusterv1.ListClustersResponse, error) {
	req := &clusterv1.ListClustersRequest{
		AccountId: accountID,
	}
	return c.ClusterService.ListClusters(c.withAuth(ctx), req)
}

// GetCluster retrieves a specific cluster
func (c *Client) GetCluster(ctx context.Context, accountID, clusterID string) (*clusterv1.GetClusterResponse, error) {
	req := &clusterv1.GetClusterRequest{
		AccountId: accountID,
		ClusterId: clusterID,
	}
	return c.ClusterService.GetCluster(c.withAuth(ctx), req)
}

// CreateCluster creates a new cluster
func (c *Client) CreateCluster(ctx context.Context, req *clusterv1.CreateClusterRequest) (*clusterv1.CreateClusterResponse, error) {
	return c.ClusterService.CreateCluster(c.withAuth(ctx), req)
}

// UpdateCluster updates an existing cluster
func (c *Client) UpdateCluster(ctx context.Context, req *clusterv1.UpdateClusterRequest) (*clusterv1.UpdateClusterResponse, error) {
	return c.ClusterService.UpdateCluster(c.withAuth(ctx), req)
}

// DeleteCluster deletes a cluster
func (c *Client) DeleteCluster(ctx context.Context, accountID, clusterID string) (*clusterv1.DeleteClusterResponse, error) {
	req := &clusterv1.DeleteClusterRequest{
		AccountId: accountID,
		ClusterId: clusterID,
	}
	return c.ClusterService.DeleteCluster(c.withAuth(ctx), req)
}

// SuspendCluster suspends a cluster
func (c *Client) SuspendCluster(ctx context.Context, accountID, clusterID string) (*clusterv1.SuspendClusterResponse, error) {
	req := &clusterv1.SuspendClusterRequest{
		AccountId: accountID,
		ClusterId: clusterID,
	}
	return c.ClusterService.SuspendCluster(c.withAuth(ctx), req)
}

// UnsuspendCluster resumes a suspended cluster
func (c *Client) UnsuspendCluster(ctx context.Context, accountID, clusterID string) (*clusterv1.UnsuspendClusterResponse, error) {
	req := &clusterv1.UnsuspendClusterRequest{
		AccountId: accountID,
		ClusterId: clusterID,
	}
	return c.ClusterService.UnsuspendCluster(c.withAuth(ctx), req)
}

// ListPackages lists available packages for a region
func (c *Client) ListPackages(ctx context.Context, accountID, cloudProviderID, cloudProviderRegionID string) (*bookingv1.ListPackagesResponse, error) {
	req := &bookingv1.ListPackagesRequest{
		AccountId:             accountID,
		CloudProviderId:       cloudProviderID,
		CloudProviderRegionId: &cloudProviderRegionID,
	}
	return c.BookingService.ListPackages(c.withAuth(ctx), req)
}

// CreateDatabaseApiKey creates a new database API key for a cluster
func (c *Client) CreateDatabaseApiKey(ctx context.Context, req *authv2.CreateDatabaseApiKeyRequest) (*authv2.CreateDatabaseApiKeyResponse, error) {
	return c.DatabaseKeyService.CreateDatabaseApiKey(c.withAuth(ctx), req)
}

// DeleteDatabaseApiKey deletes a database API key
func (c *Client) DeleteDatabaseApiKey(ctx context.Context, accountID, clusterID, keyID string) (*authv2.DeleteDatabaseApiKeyResponse, error) {
	req := &authv2.DeleteDatabaseApiKeyRequest{
		AccountId:        accountID,
		ClusterId:        clusterID,
		DatabaseApiKeyId: keyID,
	}
	return c.DatabaseKeyService.DeleteDatabaseApiKey(c.withAuth(ctx), req)
}

// Helper functions

// ConvertStorageTier converts our API storage tier to protobuf enum
func ConvertStorageTier(tier string) commonv1.StorageTierType {
	switch tier {
	case "balanced":
		return commonv1.StorageTierType_STORAGE_TIER_TYPE_BALANCED
	case "performance":
		return commonv1.StorageTierType_STORAGE_TIER_TYPE_PERFORMANCE
	case "cost-optimized":
		fallthrough
	default:
		return commonv1.StorageTierType_STORAGE_TIER_TYPE_COST_OPTIMISED
	}
}

// ConvertPhaseToString converts protobuf phase enum to string
func ConvertPhaseToString(phase clusterv1.ClusterPhase) string {
	switch phase {
	case clusterv1.ClusterPhase_CLUSTER_PHASE_HEALTHY:
		return "Healthy"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_CREATING:
		return "Creating"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_UPDATING:
		return "Updating"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_SCALING:
		return "Scaling"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_UPGRADING:
		return "Upgrading"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_SUSPENDING:
		return "Suspending"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_SUSPENDED:
		return "Suspended"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_RESUMING:
		return "Resuming"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_DELETING:
		return "Deleting"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_CREATE:
		return "FailedToCreate"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_UPDATE:
		return "FailedToUpdate"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_SUSPEND:
		return "FailedToSuspend"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_RESUME:
		return "FailedToResume"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_NOT_READY:
		return "NotReady"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_RECOVERY_MODE:
		return "RecoveryMode"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_MANUAL_MAINTENANCE:
		return "ManualMaintenance"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_SYNC:
		return "FailedToSync"
	case clusterv1.ClusterPhase_CLUSTER_PHASE_NOT_FOUND:
		return "NotFound"
	default:
		return "Unknown"
	}
}

// IsHealthyPhase checks if the phase indicates a healthy cluster
func IsHealthyPhase(phase clusterv1.ClusterPhase) bool {
	return phase == clusterv1.ClusterPhase_CLUSTER_PHASE_HEALTHY
}

// IsTransitionalPhase checks if the phase indicates an in-progress operation
func IsTransitionalPhase(phase clusterv1.ClusterPhase) bool {
	return phase == clusterv1.ClusterPhase_CLUSTER_PHASE_CREATING ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_UPDATING ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_SCALING ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_UPGRADING ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_SUSPENDING ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_RESUMING
}

// IsFailedPhase checks if the phase indicates a failure
func IsFailedPhase(phase clusterv1.ClusterPhase) bool {
	return phase == clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_CREATE ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_UPDATE ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_SUSPEND ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_RESUME ||
		phase == clusterv1.ClusterPhase_CLUSTER_PHASE_FAILED_TO_SYNC
}
