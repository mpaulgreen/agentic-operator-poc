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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachev1alpha1 "github.com/example/redis-operator/api/v1alpha1"
)

var _ = Describe("RedisCluster Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	// ============================================================
	// Lifecycle Tests
	// ============================================================
	Context("When reconciling a RedisCluster", func() {
		var (
			ctx        context.Context
			name       string
			namespace  string
			key        types.NamespacedName
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1alpha1.StorageSpec{
						Size: "10Gi",
					},
				},
			}

			reconciler = &RedisClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &cachev1alpha1.RedisCluster{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				// Remove finalizer to allow cleanup
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should add finalizer on first reconciliation", func() {
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())

			updated := &cachev1alpha1.RedisCluster{}
			Expect(k8sClient.Get(ctx, key, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement("cache.redis.example.com/finalizer"))
		})

		It("should create all managed resources", func() {
			// Multiple reconciliations to ensure all resources are created
			for i := 0; i < 3; i++ {
				_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			}

			// Verify Secret
			secret := &corev1.Secret{}
			secretKey := types.NamespacedName{Name: fmt.Sprintf("%s-auth", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, secretKey, secret)).To(Succeed())

			// Verify ConfigMap
			configMap := &corev1.ConfigMap{}
			configMapKey := types.NamespacedName{Name: fmt.Sprintf("%s-config", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, configMapKey, configMap)).To(Succeed())

			// Verify Headless Service
			headlessSvc := &corev1.Service{}
			headlessKey := types.NamespacedName{Name: fmt.Sprintf("%s-headless", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, headlessKey, headlessSvc)).To(Succeed())

			// Verify Client Service
			clientSvc := &corev1.Service{}
			clientKey := types.NamespacedName{Name: fmt.Sprintf("%s-client", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, clientKey, clientSvc)).To(Succeed())

			// Verify StatefulSet
			statefulSet := &appsv1.StatefulSet{}
			stsKey := types.NamespacedName{Name: name, Namespace: namespace}
			Expect(k8sClient.Get(ctx, stsKey, statefulSet)).To(Succeed())

			// Verify PDB (replicas=3 > 1, so PDB should be created)
			pdb := &policyv1.PodDisruptionBudget{}
			pdbKey := types.NamespacedName{Name: fmt.Sprintf("%s-pdb", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, pdbKey, pdb)).To(Succeed())
		})

		It("should be idempotent on repeated reconciliation", func() {
			// First reconcile creates everything
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile should succeed without errors
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())
			_ = result
		})

		It("should handle deletion with finalizer cleanup", func() {
			// First reconcile to add finalizer
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was added
			updated := &cachev1alpha1.RedisCluster{}
			Expect(k8sClient.Get(ctx, key, updated)).To(Succeed())
			Expect(updated.Finalizers).NotTo(BeEmpty())

			// Delete the resource
			Expect(k8sClient.Delete(ctx, updated)).To(Succeed())

			// Reconcile should handle deletion and remove finalizer
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was removed (resource may or may not still exist)
			deleted := &cachev1alpha1.RedisCluster{}
			err = k8sClient.Get(ctx, key, deleted)
			if err == nil {
				Expect(deleted.Finalizers).To(BeEmpty())
			}
			// If err != nil, resource was already garbage collected -- expected
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileSecret
	// ============================================================
	Context("When reconciling Secret", func() {
		var (
			ctx        context.Context
			name       string
			namespace  string
			key        types.NamespacedName
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1alpha1.StorageSpec{
						Size: "10Gi",
					},
				},
			}

			reconciler = &RedisClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			// Re-fetch to get UID for owner references
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &cachev1alpha1.RedisCluster{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should create Secret with REDIS_PASSWORD key when absent", func() {
			Expect(reconciler.reconcileSecret(ctx, cr)).To(Succeed())

			secret := &corev1.Secret{}
			secretKey := types.NamespacedName{Name: fmt.Sprintf("%s-auth", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, secretKey, secret)).To(Succeed())

			// Verify credential key exists
			Expect(secret.Data).To(HaveKey("REDIS_PASSWORD"))

			// Verify owner reference
			Expect(secret.OwnerReferences).To(HaveLen(1))
			Expect(secret.OwnerReferences[0].Name).To(Equal(name))

			// Verify labels
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "redis"))
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", name))
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "redis-operator"))
		})

		It("should not recreate existing Secret (idempotent)", func() {
			Expect(reconciler.reconcileSecret(ctx, cr)).To(Succeed())

			secret := &corev1.Secret{}
			secretKey := types.NamespacedName{Name: fmt.Sprintf("%s-auth", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, secretKey, secret)).To(Succeed())
			originalVersion := secret.ResourceVersion

			// Reconcile again
			Expect(reconciler.reconcileSecret(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, secretKey, secret)).To(Succeed())
			Expect(secret.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileConfigMap
	// ============================================================
	Context("When reconciling ConfigMap", func() {
		var (
			ctx        context.Context
			name       string
			namespace  string
			key        types.NamespacedName
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1alpha1.StorageSpec{
						Size: "10Gi",
					},
				},
			}

			reconciler = &RedisClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &cachev1alpha1.RedisCluster{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should create ConfigMap with redis.conf when absent", func() {
			Expect(reconciler.reconcileConfigMap(ctx, cr)).To(Succeed())

			configMap := &corev1.ConfigMap{}
			cmKey := types.NamespacedName{Name: fmt.Sprintf("%s-config", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())

			// Verify configuration content
			Expect(configMap.Data).To(HaveKey("redis.conf"))
			Expect(configMap.Data["redis.conf"]).To(ContainSubstring("maxmemory-policy"))
			Expect(configMap.Data["redis.conf"]).To(ContainSubstring("bind"))
			Expect(configMap.Data["redis.conf"]).To(ContainSubstring("port 6379"))

			// Verify owner reference
			Expect(configMap.OwnerReferences).To(HaveLen(1))
			Expect(configMap.OwnerReferences[0].Name).To(Equal(name))

			// Verify labels
			Expect(configMap.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "redis"))
			Expect(configMap.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "redis-operator"))
		})

		It("should not recreate existing ConfigMap (idempotent)", func() {
			Expect(reconciler.reconcileConfigMap(ctx, cr)).To(Succeed())

			configMap := &corev1.ConfigMap{}
			cmKey := types.NamespacedName{Name: fmt.Sprintf("%s-config", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())
			originalVersion := configMap.ResourceVersion

			// Reconcile again
			Expect(reconciler.reconcileConfigMap(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())
			Expect(configMap.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileHeadlessService
	// ============================================================
	Context("When reconciling Headless Service", func() {
		var (
			ctx        context.Context
			name       string
			namespace  string
			key        types.NamespacedName
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1alpha1.StorageSpec{
						Size: "10Gi",
					},
				},
			}

			reconciler = &RedisClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &cachev1alpha1.RedisCluster{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should create headless Service with port 6379 when absent", func() {
			Expect(reconciler.reconcileHeadlessService(ctx, cr)).To(Succeed())

			service := &corev1.Service{}
			svcKey := types.NamespacedName{Name: fmt.Sprintf("%s-headless", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, svcKey, service)).To(Succeed())

			// Verify headless (ClusterIP: None)
			Expect(service.Spec.ClusterIP).To(Equal(corev1.ClusterIPNone))

			// Verify port
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(6379)))
			Expect(service.Spec.Ports[0].Name).To(Equal("redis"))

			// Verify owner reference
			Expect(service.OwnerReferences).To(HaveLen(1))
			Expect(service.OwnerReferences[0].Name).To(Equal(name))

			// Verify selector matches labels
			Expect(service.Spec.Selector).To(HaveKeyWithValue("app.kubernetes.io/instance", name))
		})

		It("should not recreate existing headless Service (idempotent)", func() {
			Expect(reconciler.reconcileHeadlessService(ctx, cr)).To(Succeed())

			service := &corev1.Service{}
			svcKey := types.NamespacedName{Name: fmt.Sprintf("%s-headless", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, svcKey, service)).To(Succeed())
			originalVersion := service.ResourceVersion

			// Reconcile again
			Expect(reconciler.reconcileHeadlessService(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, svcKey, service)).To(Succeed())
			Expect(service.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileClientService
	// ============================================================
	Context("When reconciling Client Service", func() {
		var (
			ctx        context.Context
			name       string
			namespace  string
			key        types.NamespacedName
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1alpha1.StorageSpec{
						Size: "10Gi",
					},
				},
			}

			reconciler = &RedisClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &cachev1alpha1.RedisCluster{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should create client Service with port 6379 when absent", func() {
			Expect(reconciler.reconcileClientService(ctx, cr)).To(Succeed())

			service := &corev1.Service{}
			svcKey := types.NamespacedName{Name: fmt.Sprintf("%s-client", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, svcKey, service)).To(Succeed())

			// Verify NOT headless (regular ClusterIP)
			Expect(service.Spec.ClusterIP).NotTo(Equal(corev1.ClusterIPNone))

			// Verify port
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(6379)))
			Expect(service.Spec.Ports[0].Name).To(Equal("redis"))

			// Verify owner reference
			Expect(service.OwnerReferences).To(HaveLen(1))
			Expect(service.OwnerReferences[0].Name).To(Equal(name))

			// Verify selector matches labels
			Expect(service.Spec.Selector).To(HaveKeyWithValue("app.kubernetes.io/instance", name))
		})

		It("should not recreate existing client Service (idempotent)", func() {
			Expect(reconciler.reconcileClientService(ctx, cr)).To(Succeed())

			service := &corev1.Service{}
			svcKey := types.NamespacedName{Name: fmt.Sprintf("%s-client", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, svcKey, service)).To(Succeed())
			originalVersion := service.ResourceVersion

			// Reconcile again
			Expect(reconciler.reconcileClientService(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, svcKey, service)).To(Succeed())
			Expect(service.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileStatefulSet
	// ============================================================
	Context("When reconciling StatefulSet", func() {
		var (
			ctx        context.Context
			name       string
			namespace  string
			key        types.NamespacedName
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1alpha1.StorageSpec{
						Size: "10Gi",
					},
				},
			}

			reconciler = &RedisClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &cachev1alpha1.RedisCluster{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should create StatefulSet with correct spec when absent", func() {
			Expect(reconciler.reconcileStatefulSet(ctx, cr)).To(Succeed())

			statefulSet := &appsv1.StatefulSet{}
			stsKey := types.NamespacedName{Name: name, Namespace: namespace}
			Expect(k8sClient.Get(ctx, stsKey, statefulSet)).To(Succeed())

			// Verify replicas
			Expect(*statefulSet.Spec.Replicas).To(Equal(int32(3)))

			// Verify image contains redis
			Expect(statefulSet.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(statefulSet.Spec.Template.Spec.Containers[0].Image).To(ContainSubstring("redis"))

			// Verify headless service name
			Expect(statefulSet.Spec.ServiceName).To(Equal(fmt.Sprintf("%s-headless", name)))

			// Verify container port
			Expect(statefulSet.Spec.Template.Spec.Containers[0].Ports).To(HaveLen(1))
			Expect(statefulSet.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(6379)))

			// Verify volume mounts
			Expect(statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts).To(HaveLen(2))

			// Verify VolumeClaimTemplates
			Expect(statefulSet.Spec.VolumeClaimTemplates).To(HaveLen(1))
			Expect(statefulSet.Spec.VolumeClaimTemplates[0].Name).To(Equal("data"))

			// Verify owner reference
			Expect(statefulSet.OwnerReferences).To(HaveLen(1))
			Expect(statefulSet.OwnerReferences[0].Name).To(Equal(name))
		})

		It("should not recreate existing StatefulSet (idempotent)", func() {
			Expect(reconciler.reconcileStatefulSet(ctx, cr)).To(Succeed())

			statefulSet := &appsv1.StatefulSet{}
			stsKey := types.NamespacedName{Name: name, Namespace: namespace}
			Expect(k8sClient.Get(ctx, stsKey, statefulSet)).To(Succeed())
			originalVersion := statefulSet.ResourceVersion

			// Reconcile again -- should not update since replicas match
			Expect(reconciler.reconcileStatefulSet(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, stsKey, statefulSet)).To(Succeed())
			Expect(statefulSet.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Per-Method Tests: reconcilePodDisruptionBudget
	// ============================================================
	Context("When reconciling PodDisruptionBudget", func() {
		var (
			ctx        context.Context
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
			name       string
			namespace  string
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = fmt.Sprintf("test-pdb-%d", time.Now().UnixNano())
			name = "redis-pdb-test"

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage:  cachev1alpha1.StorageSpec{Size: "1Gi"},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cr)).To(Succeed())

			reconciler = &RedisClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}
		})

		AfterEach(func() {
			_ = k8sClient.Delete(ctx, cr)
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("should create PDB with minAvailable when replicas > 1", func() {
			Expect(reconciler.reconcilePodDisruptionBudget(ctx, cr)).To(Succeed())

			pdb := &policyv1.PodDisruptionBudget{}
			pdbKey := types.NamespacedName{Name: fmt.Sprintf("%s-pdb", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, pdbKey, pdb)).To(Succeed())

			// minAvailable = replicas - 1 = 2
			expectedMin := intstr.FromInt32(2)
			Expect(pdb.Spec.MinAvailable).To(Equal(&expectedMin))

			Expect(pdb.OwnerReferences).To(HaveLen(1))
			Expect(pdb.OwnerReferences[0].Name).To(Equal(name))

			Expect(pdb.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app.kubernetes.io/instance", name))
		})

		It("should not create PDB when replicas <= 1", func() {
			cr.Spec.Replicas = 1
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			Expect(reconciler.reconcilePodDisruptionBudget(ctx, cr)).To(Succeed())

			pdb := &policyv1.PodDisruptionBudget{}
			pdbKey := types.NamespacedName{Name: fmt.Sprintf("%s-pdb", name), Namespace: namespace}
			err := k8sClient.Get(ctx, pdbKey, pdb)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should not recreate existing PDB (idempotent)", func() {
			Expect(reconciler.reconcilePodDisruptionBudget(ctx, cr)).To(Succeed())

			pdb := &policyv1.PodDisruptionBudget{}
			pdbKey := types.NamespacedName{Name: fmt.Sprintf("%s-pdb", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, pdbKey, pdb)).To(Succeed())
			originalVersion := pdb.ResourceVersion

			Expect(reconciler.reconcilePodDisruptionBudget(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, pdbKey, pdb)).To(Succeed())
			Expect(pdb.ResourceVersion).To(Equal(originalVersion))
		})

		It("should update PDB when replicas changes", func() {
			Expect(reconciler.reconcilePodDisruptionBudget(ctx, cr)).To(Succeed())

			pdb := &policyv1.PodDisruptionBudget{}
			pdbKey := types.NamespacedName{Name: fmt.Sprintf("%s-pdb", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, pdbKey, pdb)).To(Succeed())
			originalVersion := pdb.ResourceVersion

			// Change replicas from 3 to 5, minAvailable should change from 2 to 4
			cr.Spec.Replicas = 5
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			Expect(reconciler.reconcilePodDisruptionBudget(ctx, cr)).To(Succeed())

			Expect(k8sClient.Get(ctx, pdbKey, pdb)).To(Succeed())
			Expect(pdb.ResourceVersion).NotTo(Equal(originalVersion))
			expectedMin := intstr.FromInt32(4)
			Expect(pdb.Spec.MinAvailable).To(Equal(&expectedMin))
		})
	})

	// ============================================================
	// Sentinel Tests
	// ============================================================
	Context("When reconciling Sentinel", func() {
		var (
			ctx        context.Context
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
			name       string
			namespace  string
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = fmt.Sprintf("test-sentinel-%d", time.Now().UnixNano())
			name = "redis-sentinel-test"

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3, Version: "7.4",
					Storage: cachev1alpha1.StorageSpec{Size: "1Gi"},
					Sentinel: &cachev1alpha1.SentinelSpec{
						Enabled:  true,
						Replicas: 3,
					},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cr)).To(Succeed())

			reconciler = &RedisClusterReconciler{
				Client: k8sClient, Scheme: k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}
		})

		AfterEach(func() {
			_ = k8sClient.Delete(ctx, cr)
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("should create Sentinel Deployment and Service when enabled", func() {
			Expect(reconciler.reconcileSentinel(ctx, cr)).To(Succeed())

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-sentinel", name), Namespace: namespace}, deploy)).To(Succeed())
			Expect(*deploy.Spec.Replicas).To(Equal(int32(3)))
			Expect(deploy.OwnerReferences).To(HaveLen(1))

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-sentinel", name), Namespace: namespace}, svc)).To(Succeed())
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(26379)))
		})

		It("should not create Sentinel when disabled", func() {
			cr.Spec.Sentinel = nil
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			Expect(reconciler.reconcileSentinel(ctx, cr)).To(Succeed())

			deploy := &appsv1.Deployment{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-sentinel", name), Namespace: namespace}, deploy)
			Expect(err).To(HaveOccurred())
		})

		It("should not recreate existing Sentinel (idempotent)", func() {
			Expect(reconciler.reconcileSentinel(ctx, cr)).To(Succeed())

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-sentinel", name), Namespace: namespace}, deploy)).To(Succeed())
			originalVersion := deploy.ResourceVersion

			Expect(reconciler.reconcileSentinel(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-sentinel", name), Namespace: namespace}, deploy)).To(Succeed())
			Expect(deploy.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileNetworkPolicy
	// ============================================================
	Context("When reconciling NetworkPolicy", func() {
		var (
			ctx        context.Context
			cr         *cachev1alpha1.RedisCluster
			reconciler *RedisClusterReconciler
			name       string
			namespace  string
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = fmt.Sprintf("test-np-%d", time.Now().UnixNano())
			name = "redis-np-test"

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			cr = &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: cachev1alpha1.RedisClusterSpec{
					Replicas: 3, Version: "7.4",
					Storage: cachev1alpha1.StorageSpec{Size: "1Gi"},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cr)).To(Succeed())

			reconciler = &RedisClusterReconciler{
				Client: k8sClient, Scheme: k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}
		})

		AfterEach(func() {
			_ = k8sClient.Delete(ctx, cr)
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_ = k8sClient.Delete(ctx, ns)
		})

		It("should create NetworkPolicy with ingress and egress rules", func() {
			Expect(reconciler.reconcileNetworkPolicy(ctx, cr)).To(Succeed())

			np := &networkingv1.NetworkPolicy{}
			npKey := types.NamespacedName{Name: fmt.Sprintf("%s-network-policy", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, npKey, np)).To(Succeed())

			// Verify ingress allows ports 6379 and 26379
			Expect(np.Spec.Ingress).To(HaveLen(1))
			Expect(np.Spec.Ingress[0].Ports).To(HaveLen(2))

			// Verify egress has DNS and replication rules
			Expect(np.Spec.Egress).To(HaveLen(2))

			// Verify policy types
			Expect(np.Spec.PolicyTypes).To(ContainElement(networkingv1.PolicyTypeIngress))
			Expect(np.Spec.PolicyTypes).To(ContainElement(networkingv1.PolicyTypeEgress))

			// Verify owner reference
			Expect(np.OwnerReferences).To(HaveLen(1))
			Expect(np.OwnerReferences[0].Name).To(Equal(name))
		})

		It("should not recreate existing NetworkPolicy (idempotent)", func() {
			Expect(reconciler.reconcileNetworkPolicy(ctx, cr)).To(Succeed())

			np := &networkingv1.NetworkPolicy{}
			npKey := types.NamespacedName{Name: fmt.Sprintf("%s-network-policy", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, npKey, np)).To(Succeed())
			originalVersion := np.ResourceVersion

			Expect(reconciler.reconcileNetworkPolicy(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, npKey, np)).To(Succeed())
			Expect(np.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Helper Function Tests
	// ============================================================
	Context("When testing helper functions", func() {
		It("should return correct labels with expected keys and values", func() {
			cr := &cachev1alpha1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-redis",
				},
				Spec: cachev1alpha1.RedisClusterSpec{
					Version: "7.4",
				},
			}

			labels := labelsForRedisCluster(cr)

			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/name", "redis"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/instance", "my-redis"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "redis-operator"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/part-of", "my-redis"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/version", "7.4"))
			Expect(labels).To(HaveLen(5))
		})

		It("should generate password with correct length and randomness", func() {
			password1 := generatePassword()
			password2 := generatePassword()

			// Verify length
			Expect(password1).To(HaveLen(passwordLength))
			Expect(password2).To(HaveLen(passwordLength))

			// Verify randomness (two passwords should differ)
			Expect(password1).NotTo(Equal(password2))

			// Verify characters are from the allowed charset
			for _, c := range password1 {
				Expect(passwordCharset).To(ContainSubstring(string(c)))
			}
		})
	})
})
