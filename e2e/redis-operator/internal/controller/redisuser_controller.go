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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1beta1 "github.com/example/redis-operator/api/v1beta1"
)

const (
	redisUserFinalizer = "cache.redis.example.com/redisuser-finalizer"
)

// RedisUserReconciler reconciles a RedisUser object.
type RedisUserReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisusers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisclusters,verbs=get;list;watch

func (r *RedisUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// --- PHASE 1: FETCH ---
	cr := &cachev1beta1.RedisUser{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("RedisUser resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !cr.DeletionTimestamp.IsZero() {
		return r.handleRedisUserDeletion(ctx, cr)
	}

	// --- PHASE 2: ORCHESTRATE ---
	if !controllerutil.ContainsFinalizer(cr, redisUserFinalizer) {
		controllerutil.AddFinalizer(cr, redisUserFinalizer)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	if cr.Status.Phase == "" {
		cr.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Verify parent RedisCluster exists
	cluster := &cachev1beta1.RedisCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: cr.Spec.ClusterRef, Namespace: cr.Namespace}, cluster); err != nil {
		if errors.IsNotFound(err) {
			return r.handleRedisUserError(ctx, cr, "ClusterNotFound",
				fmt.Errorf("referenced RedisCluster %q not found in namespace %s", cr.Spec.ClusterRef, cr.Namespace))
		}
		return r.handleRedisUserError(ctx, cr, "ClusterLookupFailed", err)
	}

	if err := r.reconcileUserSecret(ctx, cr); err != nil {
		return r.handleRedisUserError(ctx, cr, "SecretReconcileFailed", err)
	}

	if err := r.reconcileUserACL(ctx, cr); err != nil {
		return r.handleRedisUserError(ctx, cr, "ACLConfigMapReconcileFailed", err)
	}

	// --- PHASE 3: STATUS ---
	if err := r.updateRedisUserStatus(ctx, cr); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *RedisUserReconciler) handleRedisUserDeletion(ctx context.Context, cr *cachev1beta1.RedisUser) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(cr, redisUserFinalizer) {
		logger.Info("Performing cleanup for RedisUser", "name", cr.Name)

		r.Recorder.Event(cr, corev1.EventTypeNormal, "CleanupStarted",
			fmt.Sprintf("Cleaning up resources for RedisUser %s", cr.Name))

		controllerutil.RemoveFinalizer(cr, redisUserFinalizer)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Event(cr, corev1.EventTypeNormal, "CleanupCompleted",
			fmt.Sprintf("Cleanup completed for RedisUser %s", cr.Name))
	}

	return ctrl.Result{}, nil
}

func (r *RedisUserReconciler) handleRedisUserError(ctx context.Context, cr *cachev1beta1.RedisUser, reason string, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "Reconciliation failed", "reason", reason)
	r.Recorder.Event(cr, corev1.EventTypeWarning, reason, err.Error())

	cr.Status.Phase = "Failed"
	setRedisUserDegradedCondition(cr, reason, err.Error())
	setRedisUserUnavailableCondition(cr, reason, err.Error())
	if statusErr := r.Status().Update(ctx, cr); statusErr != nil {
		logger.Error(statusErr, "Failed to update status after error")
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1beta1.RedisUser{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
