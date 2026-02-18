/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QdrantClusterSpec defines the desired state of QdrantCluster
type QdrantClusterSpec struct {
	// AccountID is the Qdrant Cloud account ID (UUID format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	AccountID string `json:"accountID"`

	// CloudProvider specifies the cloud provider ("aws", "gcp", "azure", "hybrid")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure;hybrid
	CloudProvider string `json:"cloudProvider"`

	// CloudRegion specifies the cloud provider region (e.g., "us-east-1", "europe-west1")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	CloudRegion string `json:"cloudRegion"`

	// NodeCount specifies the number of nodes in the cluster
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	NodeCount int32 `json:"nodeCount"`

	// QdrantVersion specifies the Qdrant version (e.g., "v1.16.3" or "latest")
	// +optional
	QdrantVersion string `json:"qdrantVersion,omitempty"`

	// PackageSelection defines how to select the package for cluster nodes
	// Either packageID or resourceRequirements must be specified
	// +kubebuilder:validation:Required
	PackageSelection PackageSelection `json:"packageSelection"`

	// AdditionalDiskGiB specifies additional disk space in GiB beyond the package default
	// +optional
	AdditionalDiskGiB *int32 `json:"additionalDiskGiB,omitempty"`

	// StorageTier specifies the storage performance tier
	// +kubebuilder:validation:Enum=cost-optimized;balanced;performance
	// +kubebuilder:default=cost-optimized
	// +optional
	StorageTier StorageTierType `json:"storageTier,omitempty"`

	// Configuration allows advanced cluster configuration
	// +optional
	Configuration *ClusterConfiguration `json:"configuration,omitempty"`

	// Secret reference containing the Qdrant Cloud Management API key
	// +kubebuilder:validation:Required
	Secret SecretReference `json:"secret"`

	// ConnectionSecret specifies the name of the secret to create with connection details
	// If not specified, defaults to <cluster-name>-connection
	// +optional
	ConnectionSecret LocalObjectReference `json:"connectionSecret,omitempty"`

	// AutoCreateDatabaseKey controls whether to automatically create a database API key
	// +kubebuilder:default=true
	// +optional
	AutoCreateDatabaseKey *bool `json:"autoCreateDatabaseKey,omitempty"`

	// DatabaseKeyConfig configures the auto-created database API key
	// +optional
	DatabaseKeyConfig *DatabaseKeyConfig `json:"databaseKeyConfig,omitempty"`

	// Suspend will suspend the cluster when set to true
	// +optional
	Suspend bool `json:"suspend,omitempty"`

	// Interval is the reconciliation interval
	// +kubebuilder:default="5m"
	// +optional
	Interval *metav1.Duration `json:"interval,omitempty"`
}

// PackageSelection defines how to select the package for cluster nodes
// Either packageID or resourceRequirements must be specified, but not both
type PackageSelection struct {
	// PackageID explicitly specifies the package UUID from ListPackages API
	// +optional
	PackageID *string `json:"packageID,omitempty"`

	// ResourceRequirements specifies desired resources for automatic package selection
	// The controller will find the smallest package matching these requirements
	// +optional
	ResourceRequirements *ResourceRequirements `json:"resourceRequirements,omitempty"`
}

// ResourceRequirements defines minimum resource requirements for automatic package selection
type ResourceRequirements struct {
	// RAM specifies minimum RAM (e.g., "8GiB", "16GiB")
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i?B$`
	// +optional
	RAM string `json:"ram,omitempty"`

	// CPU specifies minimum CPU (e.g., "2000m" for 2 vCPU, "4000m" for 4 vCPU)
	// +kubebuilder:validation:Pattern=`^[0-9]+m?$`
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Disk specifies minimum disk space (e.g., "32GiB", "64GiB")
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i?B$`
	// +optional
	Disk string `json:"disk,omitempty"`
}

// StorageTierType defines storage performance tiers
// +kubebuilder:validation:Enum=cost-optimized;balanced;performance
type StorageTierType string

const (
	StorageTierCostOptimized StorageTierType = "cost-optimized"
	StorageTierBalanced      StorageTierType = "balanced"
	StorageTierPerformance   StorageTierType = "performance"
)

// ClusterConfiguration allows advanced cluster configuration options
type ClusterConfiguration struct {
	// DatabaseConfiguration for Qdrant-specific settings
	// +optional
	DatabaseConfiguration map[string]string `json:"databaseConfiguration,omitempty"`

	// Labels to apply to the cluster (max 10)
	// +optional
	// +kubebuilder:validation:MaxProperties=10
	Labels map[string]string `json:"labels,omitempty"`
}

// DatabaseKeyConfig configures the auto-created database API key
type DatabaseKeyConfig struct {
	// Name of the database API key, defaults to "<cluster-name>-key"
	// +optional
	Name string `json:"name,omitempty"`

	// Scopes defines the permissions for the key (e.g., ["read", "write"])
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// ExpiresInDays sets the expiration time in days, defaults to 90
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=365
	// +kubebuilder:default=90
	// +optional
	ExpiresInDays *int32 `json:"expiresInDays,omitempty"`
}

// QdrantClusterStatus defines the observed state of QdrantCluster
type QdrantClusterStatus struct {
	// ClusterID is the Qdrant Cloud cluster ID (UUID)
	// +optional
	ClusterID string `json:"clusterID,omitempty"`

	// Phase represents the current phase of the cluster
	// +optional
	Phase string `json:"phase,omitempty"`

	// Version is the currently running Qdrant version
	// +optional
	Version string `json:"version,omitempty"`

	// NodesUp indicates the number of nodes currently running
	// +optional
	NodesUp int32 `json:"nodesUp,omitempty"`

	// Endpoint contains the cluster connection information
	// +optional
	Endpoint *ClusterEndpoint `json:"endpoint,omitempty"`

	// PackageID is the resolved package UUID being used
	// +optional
	PackageID string `json:"packageID,omitempty"`

	// DatabaseKeyID is the ID of the auto-created database API key
	// +optional
	DatabaseKeyID string `json:"databaseKeyID,omitempty"`

	// ConnectionSecret is the name of the secret containing connection details
	// +optional
	ConnectionSecret string `json:"connectionSecret,omitempty"`

	// ObservedGeneration is the last observed generation
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the cluster's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ClusterEndpoint contains the connection information for the cluster
type ClusterEndpoint struct {
	// URL is the base URL without port (e.g., "https://abc123.eu-central-1.aws.cloud.qdrant.io")
	// +optional
	URL string `json:"url,omitempty"`

	// RESTPort is the REST API port (typically 6333)
	// +optional
	RESTPort int32 `json:"restPort,omitempty"`

	// GRPCPort is the gRPC API port (typically 6334)
	// +optional
	GRPCPort int32 `json:"grpcPort,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=qc
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.version`
// +kubebuilder:printcolumn:name="Nodes",type=integer,JSONPath=`.status.nodesUp`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// QdrantCluster is the Schema for the qdrantclusters API
type QdrantCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QdrantClusterSpec   `json:"spec,omitempty"`
	Status QdrantClusterStatus `json:"status,omitempty"`
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *QdrantCluster) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// SetReadyCondition sets the Ready condition
func (in *QdrantCluster) SetReadyCondition(status metav1.ConditionStatus, reason, message string) {
	setResourceCondition(in, ConditionReady, status, reason, message, in.Generation)
}

// SetReconcilingCondition sets the Reconciling condition
func (in *QdrantCluster) SetReconcilingCondition(status metav1.ConditionStatus, reason, message string) {
	setResourceCondition(in, ConditionReconciling, status, reason, message, in.Generation)
}

// SetSuspendedCondition sets the Suspended condition
func (in *QdrantCluster) SetSuspendedCondition(status metav1.ConditionStatus, reason, message string) {
	setResourceCondition(in, ConditionSuspended, status, reason, message, in.Generation)
}

// +kubebuilder:object:root=true

// QdrantClusterList contains a list of QdrantCluster
type QdrantClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QdrantCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QdrantCluster{}, &QdrantClusterList{})
}
