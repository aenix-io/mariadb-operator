package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	mariadbv1alpha1 "github.com/mariadb-operator/mariadb-operator/api/v1alpha1"
	condition "github.com/mariadb-operator/mariadb-operator/pkg/condition"
	"github.com/mariadb-operator/mariadb-operator/pkg/metadata"
	"github.com/mariadb-operator/mariadb-operator/pkg/pod"
	"github.com/mariadb-operator/mariadb-operator/pkg/predicate"
	"github.com/mariadb-operator/mariadb-operator/pkg/refresolver"
	sqlClient "github.com/mariadb-operator/mariadb-operator/pkg/sql"
	sqlClientSet "github.com/mariadb-operator/mariadb-operator/pkg/sqlset"
	"github.com/mariadb-operator/mariadb-operator/pkg/statefulset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type StatefulSetGaleraReconciler struct {
	client.Client
	Recorder    record.EventRecorder
	RefResolver *refresolver.RefResolver
}

//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *StatefulSetGaleraReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var sts appsv1.StatefulSet
	if err := r.Get(ctx, req.NamespacedName, &sts); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	mariadb, err := r.RefResolver.MariaDBFromAnnotation(ctx, sts.ObjectMeta)
	if err != nil {
		if errors.Is(err, refresolver.ErrMariaDBAnnotationNotFound) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !mariadb.Galera().Enabled || !mariadb.Galera().Recovery.Enabled ||
		!mariadb.HasGaleraConfiguredCondition() || mariadb.HasGaleraNotReadyCondition() {
		return ctrl.Result{}, nil
	}
	logger := log.FromContext(ctx).WithName("galera").WithName("health")
	logger.Info("Checking Galera cluster health")

	healthyCtx, cancelHealthy := context.WithTimeout(ctx, mariadb.Galera().Recovery.ClusterHealthyTimeout.Duration)
	defer cancelHealthy()
	healthy, err := r.pollUntilHealthyWithTimeout(healthyCtx, mariadb, &sts, logger)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error polling MariaDB health: %v", err)
	}

	if healthy {
		return ctrl.Result{}, nil
	}
	logger.Info("Galera cluster is not healthy")
	r.Recorder.Event(mariadb, corev1.EventTypeWarning, mariadbv1alpha1.ReasonGaleraClusterNotHealthy, "Galera cluster is not healthy")

	if err := r.patchStatus(ctx, mariadb, func(status *mariadbv1alpha1.MariaDBStatus) {
		status.GaleraRecovery = nil
		condition.SetGaleraNotReady(status, mariadb)
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("error patching MariaDB: %v", err)
	}

	return ctrl.Result{}, nil
}

func (r *StatefulSetGaleraReconciler) pollUntilHealthyWithTimeout(ctx context.Context, mariadb *mariadbv1alpha1.MariaDB,
	sts *appsv1.StatefulSet, logger logr.Logger) (bool, error) {
	err := wait.PollUntilContextCancel(ctx, 1*time.Second, true, func(ctx context.Context) (bool, error) {
		return r.isHealthy(ctx, mariadb, sts, logger)
	})
	if err != nil {
		if wait.Interrupted(err) {
			return false, nil
		}
		return false, fmt.Errorf("error polling health: %v", err)
	}
	return true, nil
}

func (r *StatefulSetGaleraReconciler) isHealthy(ctx context.Context, mariadb *mariadbv1alpha1.MariaDB, sts *appsv1.StatefulSet,
	logger logr.Logger) (bool, error) {
	logger.V(1).Info("StatefulSet ready replicas", "replicas", sts.Status.ReadyReplicas)
	if sts.Status.ReadyReplicas == mariadb.Spec.Replicas {
		return true, nil
	}
	if sts.Status.ReadyReplicas == 0 {
		return false, nil
	}

	clientCtx, cancelClient := context.WithTimeout(ctx, 5*time.Second)
	defer cancelClient()

	clientSet := sqlClientSet.NewClientSet(mariadb, r.RefResolver)
	defer clientSet.Close()
	client, err := r.readyClient(clientCtx, mariadb, clientSet)
	if err != nil {
		return false, fmt.Errorf("error getting ready client: %v", err)
	}

	status, err := client.GaleraClusterStatus(clientCtx)
	if err != nil {
		return false, fmt.Errorf("error getting Galera cluster status: %v", err)
	}
	logger.V(1).Info("Galera cluster status", "status", status)
	if status != "Primary" {
		return false, nil
	}

	size, err := client.GaleraClusterSize(clientCtx)
	if err != nil {
		return false, fmt.Errorf("error getting Galera cluster size: %v", err)
	}
	logger.V(1).Info("Galera cluster size", "size", size)
	if size != int(mariadb.Spec.Replicas) {
		return false, nil
	}

	return true, nil
}

func (r *StatefulSetGaleraReconciler) readyClient(ctx context.Context, mariadb *mariadbv1alpha1.MariaDB,
	clientSet *sqlClientSet.ClientSet) (*sqlClient.Client, error) {
	for i := 0; i < int(mariadb.Spec.Replicas); i++ {
		key := types.NamespacedName{
			Name:      statefulset.PodName(mariadb.ObjectMeta, i),
			Namespace: mariadb.Namespace,
		}
		var p corev1.Pod
		if err := r.Get(ctx, key, &p); err != nil {
			return nil, fmt.Errorf("error getting Pod: %v", err)
		}
		if !pod.PodReady(&p) {
			continue
		}

		if client, err := clientSet.ClientForIndex(ctx, i); err == nil {
			return client, nil
		}
	}
	return nil, errors.New("no Ready Pods were found")
}

func (r *StatefulSetGaleraReconciler) patchStatus(ctx context.Context, mariadb *mariadbv1alpha1.MariaDB,
	patcher func(*mariadbv1alpha1.MariaDBStatus)) error {
	patch := client.MergeFrom(mariadb.DeepCopy())
	patcher(&mariadb.Status)
	return r.Status().Patch(ctx, mariadb, patch)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StatefulSetGaleraReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		WithEventFilter(
			predicate.PredicateWithAnnotations(
				[]string{
					metadata.MariadbAnnotation,
					metadata.GaleraAnnotation,
				},
			),
		).
		Complete(r)
}
