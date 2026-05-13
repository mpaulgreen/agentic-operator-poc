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

package backup

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1beta1 "github.com/example/mongodb-operator/api/backup/v1beta1"
	databasev1beta1 "github.com/example/mongodb-operator/api/v1beta1"
)

const (
	backupPolicyFinalizer = "backup.mongodb.example.com/finalizer"
)

// MongoBackupPolicyReconciler reconciles a MongoBackupPolicy object.
type MongoBackupPolicyReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=backup.mongodb.example.com,resources=mongobackuppolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup.mongodb.example.com,resources=mongobackuppolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup.mongodb.example.com,resources=mongobackuppolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=database.mongodb.example.com,resources=mongoclusters,verbs=get;list;watch

func (r *MongoBackupPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// --- PHASE 1: FETCH ---
	cr := &backupv1beta1.MongoBackupPolicy{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("MongoBackupPolicy resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !cr.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, cr)
	}

	// --- PHASE 2: ORCHESTRATE ---
	if !controllerutil.ContainsFinalizer(cr, backupPolicyFinalizer) {
		controllerutil.AddFinalizer(cr, backupPolicyFinalizer)
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

	// Verify parent MongoCluster exists
	cluster := &databasev1beta1.MongoCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: cr.Spec.ClusterRef, Namespace: cr.Namespace}, cluster); err != nil {
		if errors.IsNotFound(err) {
			return r.handleError(ctx, cr, "ClusterNotFound",
				fmt.Errorf("referenced MongoCluster %q not found in namespace %s", cr.Spec.ClusterRef, cr.Namespace))
		}
		return r.handleError(ctx, cr, "ClusterLookupFailed", err)
	}

	if err := r.reconcileBackupPVC(ctx, cr); err != nil {
		return r.handleError(ctx, cr, "PVCReconcileFailed", err)
	}

	if err := r.reconcilePolicyCronJob(ctx, cr); err != nil {
		return r.handleError(ctx, cr, "CronJobReconcileFailed", err)
	}

	// --- PHASE 3: STATUS ---
	if err := r.updateBackupPolicyStatus(ctx, cr); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *MongoBackupPolicyReconciler) handleDeletion(ctx context.Context, cr *backupv1beta1.MongoBackupPolicy) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(cr, backupPolicyFinalizer) {
		logger.Info("Performing cleanup for MongoBackupPolicy", "name", cr.Name)

		r.Recorder.Event(cr, corev1.EventTypeNormal, "CleanupStarted",
			fmt.Sprintf("Cleaning up resources for MongoBackupPolicy %s", cr.Name))

		if err := r.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(cr, backupPolicyFinalizer)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Event(cr, corev1.EventTypeNormal, "CleanupCompleted",
			fmt.Sprintf("Cleanup completed for MongoBackupPolicy %s", cr.Name))
	}

	return ctrl.Result{}, nil
}

func (r *MongoBackupPolicyReconciler) handleError(ctx context.Context, cr *backupv1beta1.MongoBackupPolicy, reason string, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "Reconciliation failed", "reason", reason)
	r.Recorder.Event(cr, corev1.EventTypeWarning, reason, err.Error())

	cr.Status.Phase = "Failed"
	setDegradedCondition(cr, reason, err.Error())
	if statusErr := r.Status().Update(ctx, cr); statusErr != nil {
		logger.Error(statusErr, "Failed to update status after error")
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoBackupPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1beta1.MongoBackupPolicy{}).
		Owns(&batchv1.CronJob{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
