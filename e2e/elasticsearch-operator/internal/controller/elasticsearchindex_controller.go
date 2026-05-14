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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	searchv1beta1 "github.com/example/elasticsearch-operator/api/v1beta1"
)

const (
	elasticsearchIndexFinalizer = "search.elasticsearch.example.com/index-finalizer"
)

// ElasticsearchIndexReconciler reconciles a ElasticsearchIndex object.
type ElasticsearchIndexReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=search.elasticsearch.example.com,resources=elasticsearchindices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=search.elasticsearch.example.com,resources=elasticsearchindices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=search.elasticsearch.example.com,resources=elasticsearchindices/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *ElasticsearchIndexReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cr := &searchv1beta1.ElasticsearchIndex{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ElasticsearchIndex resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(cr, elasticsearchIndexFinalizer) {
			logger.Info("Performing cleanup for ElasticsearchIndex", "name", cr.Name)
			r.Recorder.Event(cr, corev1.EventTypeNormal, "CleanupStarted",
				fmt.Sprintf("Cleaning up resources for ElasticsearchIndex %s", cr.Name))

			if err := r.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(cr, elasticsearchIndexFinalizer)
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}

			r.Recorder.Event(cr, corev1.EventTypeNormal, "CleanupCompleted",
				fmt.Sprintf("Cleanup completed for ElasticsearchIndex %s", cr.Name))
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(cr, elasticsearchIndexFinalizer) {
		controllerutil.AddFinalizer(cr, elasticsearchIndexFinalizer)
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

	if err := r.reconcileIndexConfigMap(ctx, cr); err != nil {
		cr.Status.Phase = "Failed"
		r.Recorder.Event(cr, corev1.EventTypeWarning, "ConfigMapReconcileFailed", err.Error())
		setIndexCondition(cr, "Available", metav1.ConditionFalse, "ConfigMapFailed", err.Error())
		_ = r.Status().Update(ctx, cr)
		return ctrl.Result{}, err
	}

	cr.Status.Phase = "Active"
	cr.Status.IndexReady = true
	setIndexCondition(cr, "Available", metav1.ConditionTrue, "IndexConfigured", "Index template ConfigMap is ready")
	if err := r.Status().Update(ctx, cr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ElasticsearchIndexReconciler) reconcileIndexConfigMap(ctx context.Context, cr *searchv1beta1.ElasticsearchIndex) error {
	name := fmt.Sprintf("%s-index-template", cr.Name)
	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	templateJSON := fmt.Sprintf(`{
  "index_patterns": ["%s-*"],
  "settings": {
    "number_of_shards": %d,
    "number_of_replicas": %d
  }
}`, cr.Spec.IndexName, cr.Spec.Shards, cr.Spec.Replicas)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "elasticsearch",
				"app.kubernetes.io/instance":   cr.Name,
				"app.kubernetes.io/managed-by": "elasticsearch-operator",
				"app.kubernetes.io/part-of":    cr.Spec.ClusterRef,
				"app.kubernetes.io/component":  "index-template",
			},
		},
		Data: map[string]string{
			"index-template.json": templateJSON,
		},
	}

	if err := controllerutil.SetControllerReference(cr, cm, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, cm); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "ConfigMapFailed",
			fmt.Sprintf("Failed to create index ConfigMap: %v", err))
		return err
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "ConfigMapCreated",
		fmt.Sprintf("Created index template ConfigMap %s", name))
	return nil
}

func setIndexCondition(cr *searchv1beta1.ElasticsearchIndex, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i, c := range cr.Status.Conditions {
		if c.Type == conditionType {
			if c.Status != status {
				cr.Status.Conditions[i].LastTransitionTime = now
			}
			cr.Status.Conditions[i].Status = status
			cr.Status.Conditions[i].Reason = reason
			cr.Status.Conditions[i].Message = message
			cr.Status.Conditions[i].ObservedGeneration = cr.Generation
			return
		}
	}
	cr.Status.Conditions = append(cr.Status.Conditions, metav1.Condition{
		Type: conditionType, Status: status, LastTransitionTime: now,
		Reason: reason, Message: message, ObservedGeneration: cr.Generation,
	})
}

func (r *ElasticsearchIndexReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&searchv1beta1.ElasticsearchIndex{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
