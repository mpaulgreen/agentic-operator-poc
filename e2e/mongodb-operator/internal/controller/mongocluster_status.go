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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	databasev1beta1 "github.com/example/mongodb-operator/api/v1beta1"
)

func (r *MongoClusterReconciler) updateStatus(ctx context.Context, cr *databasev1beta1.MongoCluster) error {
	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, sts); err != nil {
		return err
	}

	cr.Status.ReadyReplicas = sts.Status.ReadyReplicas
	cr.Status.CurrentVersion = cr.Spec.Version
	cr.Status.PrimaryEndpoint = fmt.Sprintf("%s-client.%s.svc.cluster.local:27017", cr.Name, cr.Namespace)

	if sts.Status.ReadyReplicas == *sts.Spec.Replicas {
		cr.Status.Phase = "Running"
		setAvailableCondition(cr, "AllReplicasReady", "All MongoDB replica set members are ready")
		clearProgressingCondition(cr, "RolloutComplete", "Rollout completed")
		clearDegradedCondition(cr, "AllHealthy", "No issues detected")
	} else if sts.Status.ReadyReplicas > 0 {
		cr.Status.Phase = "Degraded"
		setUnavailableCondition(cr, "PartiallyReady",
			fmt.Sprintf("%d/%d replicas ready", sts.Status.ReadyReplicas, *sts.Spec.Replicas))
		setDegradedCondition(cr, "PartiallyReady", "Not all replicas are ready")
	} else {
		cr.Status.Phase = "Initializing"
		setUnavailableCondition(cr, "NotReady", "No replicas ready yet")
		setProgressingCondition(cr, "Initializing", "Waiting for replicas to start")
	}

	if cr.Spec.Backup != nil && cr.Spec.Backup.Enabled {
		jobList := &batchv1.JobList{}
		labelSelector := labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/instance":  cr.Name,
			"app.kubernetes.io/component": "backup",
		})
		if err := r.List(ctx, jobList, &client.ListOptions{
			Namespace:     cr.Namespace,
			LabelSelector: labelSelector,
		}); err == nil {
			for i := range jobList.Items {
				job := &jobList.Items[i]
				if job.Status.Succeeded > 0 && job.Status.CompletionTime != nil {
					if cr.Status.LastBackupTime == nil || job.Status.CompletionTime.After(cr.Status.LastBackupTime.Time) {
						cr.Status.LastBackupTime = job.Status.CompletionTime
					}
					setBackupReadyCondition(cr, "BackupComplete",
						fmt.Sprintf("Last backup completed at %s", job.Status.CompletionTime.Format("2006-01-02T15:04:05Z")))
				}
			}
		}
		if cr.Status.LastBackupTime == nil {
			clearBackupReadyCondition(cr, "NoBackupYet", "No backup has completed yet")
		}
	} else {
		clearBackupReadyCondition(cr, "BackupDisabled", "Backup is not enabled")
	}

	return r.Status().Update(ctx, cr)
}
