/*
Copyright 2026.

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

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/example/redis-operator/api/v1alpha1"
)

const (
	// redisClusterFinalizer is the finalizer added to RedisCluster resources.
	redisClusterFinalizer = "cache.redis.example.com/finalizer"
)

// RedisClusterReconciler reconciles a RedisCluster object.
type RedisClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RedisClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// --- PHASE 1: FETCH ---
	cr := &cachev1alpha1.RedisCluster{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("RedisCluster resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !cr.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, cr)
	}

	// --- PHASE 2: ORCHESTRATE ---
	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(cr, redisClusterFinalizer) {
		controllerutil.AddFinalizer(cr, redisClusterFinalizer)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set phase to Initializing if empty
	if cr.Status.Phase == "" {
		cr.Status.Phase = "Initializing"
		if err := r.Status().Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile dependent resources in dependency order
	if err := r.reconcileSecret(ctx, cr); err != nil {
		return r.handleError(ctx, cr, "SecretReconcileFailed", err)
	}

	if err := r.reconcileConfigMap(ctx, cr); err != nil {
		return r.handleError(ctx, cr, "ConfigMapReconcileFailed", err)
	}

	if err := r.reconcileHeadlessService(ctx, cr); err != nil {
		return r.handleError(ctx, cr, "HeadlessServiceReconcileFailed", err)
	}

	if err := r.reconcileClientService(ctx, cr); err != nil {
		return r.handleError(ctx, cr, "ClientServiceReconcileFailed", err)
	}

	if err := r.reconcileStatefulSet(ctx, cr); err != nil {
		return r.handleError(ctx, cr, "StatefulSetReconcileFailed", err)
	}

	if err := r.reconcilePodDisruptionBudget(ctx, cr); err != nil {
		return r.handleError(ctx, cr, "PDBReconcileFailed", err)
	}

	// --- PHASE 3: STATUS ---
	if err := r.updateStatus(ctx, cr); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// handleDeletion handles the deletion of a RedisCluster resource.
func (r *RedisClusterReconciler) handleDeletion(ctx context.Context, cr *cachev1alpha1.RedisCluster) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(cr, redisClusterFinalizer) {
		logger.Info("Performing cleanup for RedisCluster", "name", cr.Name)

		// Perform external cleanup here (e.g., flush data, revoke credentials)
		r.Recorder.Event(cr, corev1.EventTypeNormal, "CleanupStarted",
			fmt.Sprintf("Cleaning up resources for RedisCluster %s", cr.Name))

		// Remove finalizer to allow deletion to proceed
		controllerutil.RemoveFinalizer(cr, redisClusterFinalizer)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Event(cr, corev1.EventTypeNormal, "CleanupCompleted",
			fmt.Sprintf("Cleanup completed for RedisCluster %s", cr.Name))
	}

	return ctrl.Result{}, nil
}

// handleError records a warning event and returns a requeue result with the error.
func (r *RedisClusterReconciler) handleError(ctx context.Context, cr *cachev1alpha1.RedisCluster, reason string, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "Reconciliation failed", "reason", reason)
	r.Recorder.Event(cr, corev1.EventTypeWarning, reason, err.Error())

	// Set Degraded condition
	setDegradedCondition(cr, reason, err.Error())
	if statusErr := r.Status().Update(ctx, cr); statusErr != nil {
		logger.Error(statusErr, "Failed to update status after error")
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.RedisCluster{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Complete(r)
}
