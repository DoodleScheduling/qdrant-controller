/*
Copyright 2025 Doodle.

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
package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/doodlescheduling/qdrant-controller/api/v1beta1"
	qdrantclient "github.com/doodlescheduling/qdrant-controller/pkg/qdrant/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clusterv1 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/cluster/v1"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	cfg        *rest.Config
	k8sClient  client.Client
	testEnv    *envtest.Environment
	ctx        context.Context
	cancel     context.CancelFunc
	mockClient *mockQdrantClient
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// Create mock client
	mockClient = &mockQdrantClient{}

	err = (&QdrantClusterReconciler{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("QdrantCluster"),
		Recorder: k8sManager.GetEventRecorderFor("QdrantCluster"),
	}).SetupWithManager(k8sManager, QdrantClusterReconcilerOptions{})
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// mockQdrantClient mocks the Qdrant Cloud API client for testing
type mockQdrantClient struct {
	qdrantclient.Client
}

func (m *mockQdrantClient) Close() error {
	return nil
}

func (m *mockQdrantClient) CreateCluster(ctx context.Context, req *clusterv1.CreateClusterRequest) (*clusterv1.CreateClusterResponse, error) {
	return &clusterv1.CreateClusterResponse{
		Cluster: &clusterv1.Cluster{
			Id:   "test-cluster-id",
			Name: req.Cluster.Name,
		},
	}, nil
}

func (m *mockQdrantClient) GetCluster(ctx context.Context, accountID, clusterID string) (*clusterv1.GetClusterResponse, error) {
	return &clusterv1.GetClusterResponse{
		Cluster: &clusterv1.Cluster{
			Id:   clusterID,
			Name: "test-cluster",
			State: &clusterv1.ClusterState{
				Phase: clusterv1.ClusterPhase_CLUSTER_PHASE_HEALTHY,
			},
		},
	}, nil
}

func (m *mockQdrantClient) DeleteCluster(ctx context.Context, accountID, clusterID string) (*clusterv1.DeleteClusterResponse, error) {
	return &clusterv1.DeleteClusterResponse{}, nil
}

func needsExactConditions(expected []metav1.Condition, current []metav1.Condition) error {
	var expectedConditions []string
	var currentConditions []string

	for _, expectedCondition := range expected {
		expectedConditions = append(expectedConditions, expectedCondition.Type)
		var hasCondition bool
		for _, condition := range current {
			if expectedCondition.Type == condition.Type {
				hasCondition = true

				if expectedCondition.Status != condition.Status {
					return fmt.Errorf("condition %s does not match expected status %s, current status=%s; current conditions=%#v", expectedCondition.Type, expectedCondition.Status, condition.Status, current)
				}
				if expectedCondition.Reason != condition.Reason {
					return fmt.Errorf("condition %s does not match expected reason %s, current reason=%s; current conditions=%#v", expectedCondition.Type, expectedCondition.Reason, condition.Reason, current)
				}
				if expectedCondition.Message != condition.Message {
					return fmt.Errorf("condition %s does not match expected message %s, current status=%s; current conditions=%#v", expectedCondition.Type, expectedCondition.Message, condition.Message, current)
				}
			}
		}

		if !hasCondition {
			return fmt.Errorf("missing condition %s", expectedCondition.Type)
		}
	}

	for _, condition := range current {
		currentConditions = append(currentConditions, condition.Type)
	}

	if len(expectedConditions) != len(currentConditions) {
		return fmt.Errorf("expected conditions %#v do not match, current conditions=%#v", expectedConditions, currentConditions)
	}

	return nil
}

var _ = grpc.ClientConnInterface(&mockClientConn{})

type mockClientConn struct{}

func (m *mockClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	return nil
}

func (m *mockClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}
