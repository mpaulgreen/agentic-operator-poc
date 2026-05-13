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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupv1beta1 "github.com/example/mongodb-operator/api/backup/v1beta1"
)

func (r *MongoBackupPolicyReconciler) reconcilePolicyCronJob(ctx context.Context, cr *backupv1beta1.MongoBackupPolicy) error {
	name := fmt.Sprintf("%s-cronjob", cr.Name)
	existing := &batchv1.CronJob{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		if existing.Spec.Schedule != cr.Spec.Schedule {
			existing.Spec.Schedule = cr.Spec.Schedule
			if err := r.Update(ctx, existing); err != nil {
				r.Recorder.Event(cr, corev1.EventTypeWarning, "CronJobUpdateFailed",
					fmt.Sprintf("Failed to update CronJob schedule: %v", err))
				return err
			}
			r.Recorder.Event(cr, corev1.EventTypeNormal, "CronJobUpdated",
				fmt.Sprintf("Updated CronJob %s schedule to %s", name, cr.Spec.Schedule))
		}
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForBackupPolicy(cr),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: cr.Spec.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labelsForBackupPolicy(cr),
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:    "backup",
									Image:   "registry.access.redhat.com/ubi9/ubi-micro:latest",
									Command: []string{"/bin/sleep", "5"},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(cr, cronJob, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, cronJob); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "CronJobFailed",
			fmt.Sprintf("Failed to create CronJob: %v", err))
		return err
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "CronJobCreated",
		fmt.Sprintf("Created CronJob %s with schedule %s", name, cr.Spec.Schedule))
	return nil
}

func (r *MongoBackupPolicyReconciler) reconcileBackupPVC(ctx context.Context, cr *backupv1beta1.MongoBackupPolicy) error {
	name := fmt.Sprintf("%s-storage", cr.Name)
	existing := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForBackupPolicy(cr),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(cr.Spec.StorageSize),
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(cr, pvc, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, pvc); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "PVCFailed",
			fmt.Sprintf("Failed to create backup PVC: %v", err))
		return err
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "PVCCreated",
		fmt.Sprintf("Created backup PVC %s with size %s", name, cr.Spec.StorageSize))
	return nil
}
