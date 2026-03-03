package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/doodlescheduling/qdrant-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("QdrantCluster controller", func() {
	const (
		timeout  = time.Second * 4
		interval = time.Millisecond * 600
		// Valid UUID for testing (must match accountID validation pattern)
		testAccountID = "9883b383-3c03-4556-87c0-ab32de06a0ce"
	)

	When("reconciling a suspended QdrantCluster", func() {
		clusterName := fmt.Sprintf("cluster-%s", rand.String(5))

		It("should not update the status", func() {
			By("creating a new QdrantCluster")
			ctx := context.Background()

			cluster := &v1beta1.QdrantCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: "default",
				},
				Spec: v1beta1.QdrantClusterSpec{
					AccountID:     testAccountID,
					CloudProvider: v1beta1.CloudProviderAWS,
					CloudRegion:   "us-east-1",
					NodeCount:     1,
					PackageSelection: v1beta1.PackageSelection{
						ResourceRequirements: &v1beta1.ResourceRequirements{
							RAM:  resourceQuantity("2Gi"),
							CPU:  resourceQuantity("500m"),
							Disk: resourceQuantity("8Gi"),
						},
					},
					Secret: v1beta1.SecretReference{
						Name: "test-secret",
					},
					Suspend: true,
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			By("waiting for the reconciliation")
			clusterLookupKey := types.NamespacedName{Name: clusterName, Namespace: "default"}
			reconciledCluster := &v1beta1.QdrantCluster{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, reconciledCluster)
				if err != nil {
					return false
				}

				// Suspended clusters should not be reconciled
				return len(reconciledCluster.Status.Conditions) == 0
			}, timeout, interval).Should(BeTrue())
		})
	})

	clusterName := fmt.Sprintf("cluster-%s", rand.String(5))
	secretName := fmt.Sprintf("secret-%s", rand.String(5))
	When("it can't find the referenced secret with credentials", func() {
		It("should update the status with error condition", func() {
			By("creating a new QdrantCluster")
			ctx := context.Background()

			cluster := &v1beta1.QdrantCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: "default",
				},
				Spec: v1beta1.QdrantClusterSpec{
					AccountID:     testAccountID,
					CloudProvider: v1beta1.CloudProviderAWS,
					CloudRegion:   "us-east-1",
					NodeCount:     1,
					PackageSelection: v1beta1.PackageSelection{
						ResourceRequirements: &v1beta1.ResourceRequirements{
							RAM:  resourceQuantity("2Gi"),
							CPU:  resourceQuantity("500m"),
							Disk: resourceQuantity("8Gi"),
						},
					},
					Secret: v1beta1.SecretReference{
						Name: secretName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			By("waiting for the reconciliation")
			clusterLookupKey := types.NamespacedName{Name: clusterName, Namespace: "default"}
			reconciledCluster := &v1beta1.QdrantCluster{}

			expectedStatus := &v1beta1.QdrantClusterStatus{
				Conditions: []metav1.Condition{
					{
						Type:    v1beta1.ConditionReady,
						Status:  metav1.ConditionFalse,
						Reason:  "ReconciliationFailed",
						Message: fmt.Sprintf(`failed to get secret: Secret "%s" not found`, secretName),
					},
				},
			}

			Eventually(func() error {
				err := k8sClient.Get(ctx, clusterLookupKey, reconciledCluster)
				if err != nil {
					return err
				}
				return needsExactConditions(expectedStatus.Conditions, reconciledCluster.Status.Conditions)
			}, timeout, interval).Should(Not(HaveOccurred()))
		})
	})

	clusterName2 := fmt.Sprintf("cluster-%s", rand.String(5))
	secretName2 := fmt.Sprintf("secret-%s", rand.String(5))
	When("the secret exists but is missing the apiKey field", func() {
		It("should update the status with error condition", func() {
			By("creating a secret without apiKey")
			ctx := context.Background()

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName2,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"wrongkey": []byte("some-value"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("creating a new QdrantCluster")
			cluster := &v1beta1.QdrantCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName2,
					Namespace: "default",
				},
				Spec: v1beta1.QdrantClusterSpec{
					AccountID:     testAccountID,
					CloudProvider: v1beta1.CloudProviderAWS,
					CloudRegion:   "us-east-1",
					NodeCount:     1,
					PackageSelection: v1beta1.PackageSelection{
						ResourceRequirements: &v1beta1.ResourceRequirements{
							RAM:  resourceQuantity("2Gi"),
							CPU:  resourceQuantity("500m"),
							Disk: resourceQuantity("8Gi"),
						},
					},
					Secret: v1beta1.SecretReference{
						Name: secretName2,
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			By("waiting for the reconciliation")
			clusterLookupKey := types.NamespacedName{Name: clusterName2, Namespace: "default"}
			reconciledCluster := &v1beta1.QdrantCluster{}

			expectedStatus := &v1beta1.QdrantClusterStatus{
				Conditions: []metav1.Condition{
					{
						Type:    v1beta1.ConditionReady,
						Status:  metav1.ConditionFalse,
						Reason:  "ReconciliationFailed",
						Message: "secret must contain apiKey key",
					},
				},
			}

			Eventually(func() error {
				err := k8sClient.Get(ctx, clusterLookupKey, reconciledCluster)
				if err != nil {
					return err
				}
				return needsExactConditions(expectedStatus.Conditions, reconciledCluster.Status.Conditions)
			}, timeout, interval).Should(Not(HaveOccurred()))
		})
	})
})
