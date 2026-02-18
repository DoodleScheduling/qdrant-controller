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
	"time"

	infrav1beta1 "github.com/doodlescheduling/qdrant-controller/api/v1beta1"
	qdrantclient "github.com/doodlescheduling/qdrant-controller/pkg/qdrant/client"
	"github.com/go-logr/logr"
	clusterv1 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/cluster/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//+kubebuilder:rbac:groups=qdrant.infra.doodle.com,resources=qdrantclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=qdrant.infra.doodle.com,resources=qdrantclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=qdrant.infra.doodle.com,resources=qdrantclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;delete;patch;update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

const (
	qdrantClusterFinalizer  = "qdrant.infra.doodle.com/finalizer"
	databaseKeyIDAnnotation = "qdrant.infra.doodle.com/database-key-id"
	secretIndexKey          = ".metadata.secret"
)

// QdrantClusterReconciler reconciles a QdrantCluster object
type QdrantClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

type QdrantClusterReconcilerOptions struct {
	MaxConcurrentReconciles int
}

func (r *QdrantClusterReconciler) SetupWithManager(mgr ctrl.Manager, opts QdrantClusterReconcilerOptions) error {
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &infrav1beta1.QdrantCluster{}, secretIndexKey,
		func(o client.Object) []string {
			cluster := o.(*infrav1beta1.QdrantCluster)
			keys := []string{}

			if cluster.Spec.Secret.Name != "" {
				keys = []string{
					fmt.Sprintf("%s/%s", cluster.GetNamespace(), cluster.Spec.Secret.Name),
				}
			}

			return keys
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.QdrantCluster{}, builder.WithPredicates(
			predicate.GenerationChangedPredicate{},
		)).
		WithOptions(controller.Options{MaxConcurrentReconciles: opts.MaxConcurrentReconciles}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.requestsForSecretChange),
		).
		Complete(r)
}

// objectKey returns client.ObjectKey for the object.
func objectKey(object metav1.Object) client.ObjectKey {
	return client.ObjectKey{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
}

func (r *QdrantClusterReconciler) requestsForSecretChange(ctx context.Context, o client.Object) []reconcile.Request {
	secret, ok := o.(*corev1.Secret)
	if !ok {
		panic(fmt.Sprintf("expected a Secret, got %T", o))
	}

	var list infrav1beta1.QdrantClusterList
	if err := r.List(ctx, &list, client.MatchingFields{
		secretIndexKey: objectKey(secret).String(),
	}); err != nil {
		return nil
	}

	var reqs []reconcile.Request
	for _, cluster := range list.Items {
		r.Log.V(1).Info("referenced secret from a QdrantCluster changed detected", "namespace", cluster.GetNamespace(), "name", cluster.GetName())
		reqs = append(reqs, reconcile.Request{NamespacedName: objectKey(&cluster)})
	}

	return reqs
}

func (r *QdrantClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("namespace", req.Namespace, "name", req.Name)

	cluster := infrav1beta1.QdrantCluster{}
	err := r.Get(ctx, req.NamespacedName, &cluster)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Handle deletion
	if !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, cluster, logger)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&cluster, qdrantClusterFinalizer) {
		controllerutil.AddFinalizer(&cluster, qdrantClusterFinalizer)
		if err := r.Update(ctx, &cluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if cluster.Spec.Suspend {
		logger.Info("cluster is suspended")
		return r.reconcileSuspend(ctx, cluster, logger)
	}

	logger.Info("reconciling qdrant cluster")
	cluster, result, err := r.reconcile(ctx, cluster, logger)
	cluster.Status.ObservedGeneration = cluster.GetGeneration()

	if err != nil {
		logger.Error(err, "reconcile error occurred")
		cluster.SetReadyCondition(metav1.ConditionFalse, "ReconciliationFailed", err.Error())
		r.Recorder.Event(&cluster, "Warning", "ReconciliationFailed", err.Error())
	}

	// Update status after reconciliation
	if err := r.patchStatus(ctx, &cluster); err != nil {
		logger.Error(err, "unable to update status after reconciliation")
		return ctrl.Result{Requeue: true}, err
	}

	if err == nil && cluster.Spec.Interval != nil {
		result.RequeueAfter = cluster.Spec.Interval.Duration
	}

	return result, err
}

func (r *QdrantClusterReconciler) qdrantClient(ctx context.Context, cluster infrav1beta1.QdrantCluster) (*qdrantclient.Client, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      cluster.Spec.Secret.Name,
		Namespace: cluster.Namespace,
	}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	apiKeyKey := cluster.Spec.Secret.APIKeyKey
	if apiKeyKey == "" {
		apiKeyKey = "apiKey"
	}

	apiKey := string(secret.Data[apiKeyKey])
	if apiKey == "" {
		return nil, fmt.Errorf("secret must contain %s key", apiKeyKey)
	}

	return qdrantclient.NewClient(apiKey)
}

func (r *QdrantClusterReconciler) reconcile(ctx context.Context, cluster infrav1beta1.QdrantCluster, logger logr.Logger) (infrav1beta1.QdrantCluster, ctrl.Result, error) {
	qdrant, err := r.qdrantClient(ctx, cluster)
	if err != nil {
		return cluster, reconcile.Result{}, err
	}
	defer qdrant.Close()

	connectionSecretName := fmt.Sprintf("%s-connection", cluster.Name)
	if cluster.Spec.ConnectionSecret.Name != "" {
		connectionSecretName = cluster.Spec.ConnectionSecret.Name
	}

	// If cluster already exists, check its status
	if cluster.Status.ClusterID != "" {
		qdrantCluster, err := qdrant.GetCluster(ctx, cluster.Spec.AccountID, cluster.Status.ClusterID)
		if err != nil {
			logger.Error(err, "failed to get Qdrant cluster")
			return cluster, reconcile.Result{}, err
		}

		if qdrantCluster.Cluster == nil {
			// Cluster not found, reset status
			logger.Info("cluster not found in Qdrant Cloud, will recreate")
			cluster.Status.ClusterID = ""
			cluster.Status.Phase = ""
			return cluster, reconcile.Result{Requeue: true}, nil
		}

		// Update status from cluster state
		cluster = r.updateStatusFromCluster(cluster, qdrantCluster.Cluster)

		// Handle different phases
		if qdrantCluster.Cluster.State != nil {
			switch qdrantCluster.Cluster.State.Phase {
			case clusterv1.ClusterPhase_CLUSTER_PHASE_HEALTHY:
				cluster.SetReconcilingCondition(metav1.ConditionFalse, "ReconcileComplete", "Reconciliation completed")
				cluster.SetReadyCondition(metav1.ConditionTrue, "ClusterHealthy", "Cluster is healthy")

				// Ensure database API key exists if auto-create is enabled
				autoCreate := true
				if cluster.Spec.AutoCreateDatabaseKey != nil {
					autoCreate = *cluster.Spec.AutoCreateDatabaseKey
				}

				if autoCreate && cluster.Status.DatabaseKeyID == "" {
					logger.Info("creating database API key")
					if err := r.createDatabaseKey(ctx, qdrant, &cluster, logger); err != nil {
						return cluster, reconcile.Result{}, fmt.Errorf("failed to create database key: %w", err)
					}
					return cluster, reconcile.Result{Requeue: true}, nil
				}

				// Ensure connection secret exists
				if err := r.ensureConnectionSecret(ctx, &cluster, connectionSecretName, qdrantCluster.Cluster); err != nil {
					return cluster, reconcile.Result{}, fmt.Errorf("failed to create connection secret: %w", err)
				}

			case clusterv1.ClusterPhase_CLUSTER_PHASE_CREATING,
				clusterv1.ClusterPhase_CLUSTER_PHASE_UPDATING,
				clusterv1.ClusterPhase_CLUSTER_PHASE_SCALING,
				clusterv1.ClusterPhase_CLUSTER_PHASE_UPGRADING:
				cluster.SetReconcilingCondition(metav1.ConditionTrue, "ClusterProvisioning", fmt.Sprintf("Cluster is %s", cluster.Status.Phase))
				return cluster, reconcile.Result{RequeueAfter: time.Second * 30}, nil

			default:
				if qdrantclient.IsFailedPhase(qdrantCluster.Cluster.State.Phase) {
					cluster.SetReadyCondition(metav1.ConditionFalse, "ClusterFailed", fmt.Sprintf("Cluster is in failed state: %s - %s", cluster.Status.Phase, qdrantCluster.Cluster.State.Reason))
				} else {
					cluster.SetReadyCondition(metav1.ConditionFalse, "ClusterNotReady", fmt.Sprintf("Cluster status: %s", cluster.Status.Phase))
				}
			}
		}

		// Check if cluster needs update
		if r.needsUpdate(cluster, qdrantCluster.Cluster) {
			logger.Info("updating qdrant cluster")
			if err := r.updateCluster(ctx, qdrant, &cluster, logger); err != nil {
				return cluster, reconcile.Result{}, fmt.Errorf("failed to update cluster: %w", err)
			}
			cluster.SetReconcilingCondition(metav1.ConditionTrue, "UpdatingCluster", "Updating Qdrant cluster")
			return cluster, reconcile.Result{RequeueAfter: time.Second * 30}, nil
		}

		return cluster, reconcile.Result{}, nil
	}

	// Cluster doesn't exist yet, create it
	logger.Info("creating new qdrant cluster")
	cluster.SetReconcilingCondition(metav1.ConditionTrue, "CreatingCluster", "Creating Qdrant cluster")

	if err := r.createCluster(ctx, qdrant, &cluster, logger); err != nil {
		return cluster, reconcile.Result{}, fmt.Errorf("failed to create cluster: %w", err)
	}

	r.Recorder.Event(&cluster, "Normal", "ClusterCreated", "Qdrant cluster creation initiated")
	return cluster, reconcile.Result{RequeueAfter: time.Second * 30}, nil
}

func (r *QdrantClusterReconciler) reconcileSuspend(ctx context.Context, cluster infrav1beta1.QdrantCluster, logger logr.Logger) (ctrl.Result, error) {
	if cluster.Status.ClusterID == "" {
		// No cluster to suspend
		cluster.SetSuspendedCondition(metav1.ConditionTrue, "ClusterSuspended", "Cluster creation prevented by suspend flag")
		if err := r.patchStatus(ctx, &cluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	qdrant, err := r.qdrantClient(ctx, cluster)
	if err != nil {
		return reconcile.Result{}, err
	}
	defer qdrant.Close()

	// Check current cluster status
	qdrantCluster, err := qdrant.GetCluster(ctx, cluster.Spec.AccountID, cluster.Status.ClusterID)
	if err != nil {
		return ctrl.Result{}, err
	}

	if qdrantCluster.Cluster != nil && qdrantCluster.Cluster.State != nil {
		if qdrantCluster.Cluster.State.Phase == clusterv1.ClusterPhase_CLUSTER_PHASE_SUSPENDED {
			cluster.SetSuspendedCondition(metav1.ConditionTrue, "ClusterSuspended", "Cluster is suspended")
			if err := r.patchStatus(ctx, &cluster); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		// Suspend the cluster
		logger.Info("suspending qdrant cluster")
		_, err = qdrant.SuspendCluster(ctx, cluster.Spec.AccountID, cluster.Status.ClusterID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to suspend cluster: %w", err)
		}

		cluster.SetSuspendedCondition(metav1.ConditionFalse, "ClusterSuspending", "Cluster is being suspended")
		if err := r.patchStatus(ctx, &cluster); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	return ctrl.Result{}, nil
}

func (r *QdrantClusterReconciler) reconcileDelete(ctx context.Context, cluster infrav1beta1.QdrantCluster, logger logr.Logger) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(&cluster, qdrantClusterFinalizer) {
		return ctrl.Result{}, nil
	}

	logger.Info("deleting qdrant cluster")

	qdrant, err := r.qdrantClient(ctx, cluster)
	if err != nil {
		return reconcile.Result{}, err
	}
	defer qdrant.Close()

	// Delete database API key if it exists
	if cluster.Status.DatabaseKeyID != "" && cluster.Status.ClusterID != "" {
		logger.Info("deleting database API key", "keyID", cluster.Status.DatabaseKeyID)
		_, err := qdrant.DeleteDatabaseApiKey(ctx, cluster.Spec.AccountID, cluster.Status.ClusterID, cluster.Status.DatabaseKeyID)
		if err != nil {
			logger.Error(err, "failed to delete database API key")
			// Continue with cluster deletion even if key deletion fails
		}
	}

	// Delete cluster if it exists
	if cluster.Status.ClusterID != "" {
		logger.Info("deleting cluster from Qdrant Cloud", "clusterID", cluster.Status.ClusterID)
		_, err := qdrant.DeleteCluster(ctx, cluster.Spec.AccountID, cluster.Status.ClusterID)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete cluster: %w", err)
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(&cluster, qdrantClusterFinalizer)
	if err := r.Update(ctx, &cluster); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("qdrant cluster deleted")
	return ctrl.Result{}, nil
}

func (r *QdrantClusterReconciler) patchStatus(ctx context.Context, cluster *infrav1beta1.QdrantCluster) error {
	key := client.ObjectKeyFromObject(cluster)
	latest := &infrav1beta1.QdrantCluster{}
	if err := r.Client.Get(ctx, key, latest); err != nil {
		return err
	}

	return r.Client.Status().Patch(ctx, cluster, client.MergeFrom(latest))
}

// Continue in next part...
