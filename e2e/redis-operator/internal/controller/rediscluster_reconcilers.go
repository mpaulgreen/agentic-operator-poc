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
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cachev1alpha1 "github.com/example/redis-operator/api/v1alpha1"
)

// reconcileSecret ensures the Redis authentication Secret exists.
// Secret name: <name>-auth
// Keys: REDIS_PASSWORD
func (r *RedisClusterReconciler) reconcileSecret(ctx context.Context, cr *cachev1alpha1.RedisCluster) error {
	name := fmt.Sprintf("%s-auth", cr.Name)

	// If auth uses an existing secret, skip creation
	if cr.Spec.Auth != nil && cr.Spec.Auth.ExistingSecret != "" {
		return nil
	}

	// 1. CHECK if exists
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		return nil // EXISTS -- idempotent, nothing to do
	}
	if !errors.IsNotFound(err) {
		return err // ACTUAL ERROR
	}

	// 2. BUILD desired state
	password := generatePassword()
	if cr.Spec.Auth != nil && cr.Spec.Auth.Password != "" {
		password = cr.Spec.Auth.Password
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForRedisCluster(cr),
		},
		StringData: map[string]string{
			"REDIS_PASSWORD": password,
		},
	}

	// 3. SET OWNER REFERENCE (for garbage collection)
	if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
		return err
	}

	// 4. CREATE
	if err := r.Create(ctx, secret); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "SecretFailed", err.Error())
		return err
	}

	// 5. RECORD SUCCESS EVENT
	r.Recorder.Event(cr, corev1.EventTypeNormal, "SecretCreated", name)
	return nil
}

// reconcileConfigMap ensures the Redis configuration ConfigMap exists.
// ConfigMap name: <name>-config
// Key: redis.conf with bind, protected-mode, port, maxmemory-policy settings
func (r *RedisClusterReconciler) reconcileConfigMap(ctx context.Context, cr *cachev1alpha1.RedisCluster) error {
	name := fmt.Sprintf("%s-config", cr.Name)

	// 1. CHECK if exists
	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		return nil // EXISTS -- idempotent, nothing to do
	}
	if !errors.IsNotFound(err) {
		return err // ACTUAL ERROR
	}

	// 2. BUILD desired state
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForRedisCluster(cr),
		},
		Data: map[string]string{
			"redis.conf": "bind 0.0.0.0\nprotected-mode yes\nport 6379\nmaxmemory-policy allkeys-lru\nappendonly yes\nappendfilename \"appendonly.aof\"\ndir /var/lib/redis/data\n",
		},
	}

	// 3. SET OWNER REFERENCE
	if err := controllerutil.SetControllerReference(cr, configMap, r.Scheme); err != nil {
		return err
	}

	// 4. CREATE
	if err := r.Create(ctx, configMap); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "ConfigMapFailed", err.Error())
		return err
	}

	// 5. RECORD SUCCESS EVENT
	r.Recorder.Event(cr, corev1.EventTypeNormal, "ConfigMapCreated", name)
	return nil
}

// reconcileHeadlessService ensures the headless Service exists for StatefulSet DNS.
// Service name: <name>-headless, ClusterIP None, port 6379
func (r *RedisClusterReconciler) reconcileHeadlessService(ctx context.Context, cr *cachev1alpha1.RedisCluster) error {
	name := fmt.Sprintf("%s-headless", cr.Name)

	// 1. CHECK if exists
	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		return nil // EXISTS -- idempotent, nothing to do
	}
	if !errors.IsNotFound(err) {
		return err // ACTUAL ERROR
	}

	// 2. BUILD desired state
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForRedisCluster(cr),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Selector:  labelsForRedisCluster(cr),
			Ports: []corev1.ServicePort{
				{
					Name:     "redis",
					Port:     6379,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// 3. SET OWNER REFERENCE
	if err := controllerutil.SetControllerReference(cr, service, r.Scheme); err != nil {
		return err
	}

	// 4. CREATE
	if err := r.Create(ctx, service); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "HeadlessServiceFailed", err.Error())
		return err
	}

	// 5. RECORD SUCCESS EVENT
	r.Recorder.Event(cr, corev1.EventTypeNormal, "HeadlessServiceCreated", name)
	return nil
}

// reconcileClientService ensures the client-facing Service exists.
// Service name: <name>-client, regular ClusterIP, port 6379
func (r *RedisClusterReconciler) reconcileClientService(ctx context.Context, cr *cachev1alpha1.RedisCluster) error {
	name := fmt.Sprintf("%s-client", cr.Name)

	// 1. CHECK if exists
	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		return nil // EXISTS -- idempotent, nothing to do
	}
	if !errors.IsNotFound(err) {
		return err // ACTUAL ERROR
	}

	// 2. BUILD desired state
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForRedisCluster(cr),
		},
		Spec: corev1.ServiceSpec{
			Selector: labelsForRedisCluster(cr),
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Port:       6379,
					TargetPort: intstr.FromInt32(6379),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// 3. SET OWNER REFERENCE
	if err := controllerutil.SetControllerReference(cr, service, r.Scheme); err != nil {
		return err
	}

	// 4. CREATE
	if err := r.Create(ctx, service); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "ClientServiceFailed", err.Error())
		return err
	}

	// 5. RECORD SUCCESS EVENT
	r.Recorder.Event(cr, corev1.EventTypeNormal, "ClientServiceCreated", name)
	return nil
}

// reconcileStatefulSet ensures the Redis StatefulSet exists and is up to date.
// Image: registry.redhat.io/rhel9/redis-7 (OpenShift-compatible)
// Data dir: /var/lib/redis/data, Config dir: /etc/redis/
func (r *RedisClusterReconciler) reconcileStatefulSet(ctx context.Context, cr *cachev1alpha1.RedisCluster) error {
	name := cr.Name

	// 1. CHECK if exists
	existing := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		// Check-update: reconcile all mutable spec fields
		updated := false
		if *existing.Spec.Replicas != cr.Spec.Replicas {
			existing.Spec.Replicas = &cr.Spec.Replicas
			updated = true
		}
		desiredAffinity := podAffinityForRedisCluster(cr)
		if !reflect.DeepEqual(existing.Spec.Template.Spec.Affinity, desiredAffinity) {
			existing.Spec.Template.Spec.Affinity = desiredAffinity
			updated = true
		}
		desiredImage := imageForRedisCluster(cr)
		if existing.Spec.Template.Spec.Containers[0].Image != desiredImage {
			existing.Spec.Template.Spec.Containers[0].Image = desiredImage
			updated = true
		}
		if !reflect.DeepEqual(existing.Spec.Template.Spec.Containers[0].Resources, cr.Spec.Resources) {
			existing.Spec.Template.Spec.Containers[0].Resources = cr.Spec.Resources
			updated = true
		}
		if updated {
			if err := r.Update(ctx, existing); err != nil {
				r.Recorder.Event(cr, corev1.EventTypeWarning, "StatefulSetUpdateFailed", err.Error())
				return err
			}
			r.Recorder.Event(cr, corev1.EventTypeNormal, "StatefulSetUpdated",
				fmt.Sprintf("Updated StatefulSet %s", cr.Name))
		}
		return nil
	}
	if !errors.IsNotFound(err) {
		return err // ACTUAL ERROR
	}

	// 2. BUILD desired state
	labels := labelsForRedisCluster(cr)
	replicas := cr.Spec.Replicas
	image := imageForRedisCluster(cr)
	authSecretName := fmt.Sprintf("%s-auth", cr.Name)
	configMapName := fmt.Sprintf("%s-config", cr.Name)
	headlessServiceName := fmt.Sprintf("%s-headless", cr.Name)

	// Use existing secret if specified
	if cr.Spec.Auth != nil && cr.Spec.Auth.ExistingSecret != "" {
		authSecretName = cr.Spec.Auth.ExistingSecret
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: headlessServiceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Affinity: podAffinityForRedisCluster(cr),
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: image,
							Command: []string{
								"redis-server",
								"/etc/redis/redis.conf",
								"--requirepass",
								"$(REDIS_PASSWORD)",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: 6379,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: authSecretName,
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/var/lib/redis/data",
								},
								{
									Name:      "config",
									MountPath: "/etc/redis/",
								},
							},
							Resources: cr.Spec.Resources,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "data",
						Labels: labels,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(cr.Spec.Storage.Size),
							},
						},
						StorageClassName: cr.Spec.Storage.StorageClassName,
					},
				},
			},
		},
	}

	// 3. SET OWNER REFERENCE
	if err := controllerutil.SetControllerReference(cr, statefulSet, r.Scheme); err != nil {
		return err
	}

	// 4. CREATE
	if err := r.Create(ctx, statefulSet); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "StatefulSetFailed", err.Error())
		return err
	}

	// 5. RECORD SUCCESS EVENT
	r.Recorder.Event(cr, corev1.EventTypeNormal, "StatefulSetCreated", name)
	return nil
}

// reconcilePodDisruptionBudget ensures the PDB exists when replicas > 1.
// PDB name: <name>-pdb
// Only created when spec.replicas > 1.
func (r *RedisClusterReconciler) reconcilePodDisruptionBudget(ctx context.Context, cr *cachev1alpha1.RedisCluster) error {
	name := fmt.Sprintf("%s-pdb", cr.Name)

	if cr.Spec.Replicas <= 1 {
		existing := &policyv1.PodDisruptionBudget{}
		err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
		if err == nil {
			if err := r.Delete(ctx, existing); err != nil {
				r.Recorder.Event(cr, corev1.EventTypeWarning, "PDBDeleteFailed", err.Error())
				return err
			}
			r.Recorder.Event(cr, corev1.EventTypeNormal, "PDBDeleted", name)
		}
		return nil
	}

	// 1. CHECK if exists
	existing := &policyv1.PodDisruptionBudget{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		// Check-update: reconcile PDB spec if replicas changed
		desiredMinAvail := cr.Spec.Replicas - 1
		if desiredMinAvail < 1 {
			desiredMinAvail = 1
		}
		val := intstr.FromInt32(desiredMinAvail)
		if existing.Spec.MinAvailable == nil || *existing.Spec.MinAvailable != val {
			existing.Spec.MinAvailable = &val
			if err := r.Update(ctx, existing); err != nil {
				r.Recorder.Event(cr, corev1.EventTypeWarning, "PDBUpdateFailed", err.Error())
				return err
			}
			r.Recorder.Event(cr, corev1.EventTypeNormal, "PDBUpdated", name)
		}
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	// 2. BUILD desired state
	labels := labelsForRedisCluster(cr)
	minAvail := cr.Spec.Replicas - 1
	if minAvail < 1 {
		minAvail = 1
	}
	minAvailVal := intstr.FromInt32(minAvail)

	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailVal,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}

	// 3. SET OWNER REFERENCE
	if err := controllerutil.SetControllerReference(cr, pdb, r.Scheme); err != nil {
		return err
	}

	// 4. CREATE
	if err := r.Create(ctx, pdb); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "PDBFailed", err.Error())
		return err
	}

	// 5. RECORD SUCCESS EVENT
	r.Recorder.Event(cr, corev1.EventTypeNormal, "PDBCreated", name)
	return nil
}

// podAffinityForRedisCluster returns pod anti-affinity to spread replicas across nodes.
// Returns nil when replicas <= 1.
func podAffinityForRedisCluster(cr *cachev1alpha1.RedisCluster) *corev1.Affinity {
	if cr.Spec.Replicas <= 1 {
		return nil
	}

	labelSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/instance": cr.Name,
		},
	}

	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: labelSelector,
						TopologyKey:   "kubernetes.io/hostname",
					},
				},
			},
		},
	}
}
