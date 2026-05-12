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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachev1beta1 "github.com/example/redis-operator/api/v1beta1"
)

var _ = Describe("RedisUser Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	// ============================================================
	// Lifecycle Tests
	// ============================================================
	Context("When reconciling a RedisUser", func() {
		var (
			ctx           context.Context
			name          string
			clusterName   string
			namespace     string
			key           types.NamespacedName
			cr            *cachev1beta1.RedisUser
			cluster       *cachev1beta1.RedisCluster
			reconciler    *RedisUserReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-user-%d", time.Now().UnixNano())
			clusterName = fmt.Sprintf("test-cluster-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			// Create parent RedisCluster (required by controller)
			cluster = &cachev1beta1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: namespace,
				},
				Spec: cachev1beta1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1beta1.StorageSpec{
						Size: "10Gi",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			cr = &cachev1beta1.RedisUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1beta1.RedisUserSpec{
					Username:   "appuser",
					ClusterRef: clusterName,
					Permissions: []string{"+@read", "~app:*"},
				},
			}

			reconciler = &RedisUserReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		})

		AfterEach(func() {
			// Cleanup RedisUser
			resource := &cachev1beta1.RedisUser{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
			// Cleanup parent RedisCluster
			clusterRes := &cachev1beta1.RedisCluster{}
			clusterKey := types.NamespacedName{Name: clusterName, Namespace: namespace}
			if err := k8sClient.Get(ctx, clusterKey, clusterRes); err == nil {
				clusterRes.Finalizers = nil
				_ = k8sClient.Update(ctx, clusterRes)
				Expect(k8sClient.Delete(ctx, clusterRes)).To(Succeed())
			}
		})

		It("should add finalizer on first reconciliation", func() {
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())

			updated := &cachev1beta1.RedisUser{}
			Expect(k8sClient.Get(ctx, key, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement("cache.redis.example.com/redisuser-finalizer"))
		})

		It("should create all managed resources", func() {
			// Multiple reconciliations to ensure all resources are created
			for i := 0; i < 3; i++ {
				_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			}

			// Verify Secret
			secret := &corev1.Secret{}
			secretKey := types.NamespacedName{Name: fmt.Sprintf("%s-user-secret", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, secretKey, secret)).To(Succeed())

			// Verify ACL ConfigMap
			configMap := &corev1.ConfigMap{}
			cmKey := types.NamespacedName{Name: fmt.Sprintf("%s-acl", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())
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
			updated := &cachev1beta1.RedisUser{}
			Expect(k8sClient.Get(ctx, key, updated)).To(Succeed())
			Expect(updated.Finalizers).NotTo(BeEmpty())

			// Delete the resource
			Expect(k8sClient.Delete(ctx, updated)).To(Succeed())

			// Reconcile should handle deletion and remove finalizer
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was removed (resource may or may not still exist)
			deleted := &cachev1beta1.RedisUser{}
			err = k8sClient.Get(ctx, key, deleted)
			if err == nil {
				Expect(deleted.Finalizers).To(BeEmpty())
			}
			// If err != nil, resource was already garbage collected -- expected
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileUserSecret
	// ============================================================
	Context("When reconciling UserSecret", func() {
		var (
			ctx         context.Context
			name        string
			clusterName string
			namespace   string
			key         types.NamespacedName
			cr          *cachev1beta1.RedisUser
			cluster     *cachev1beta1.RedisCluster
			reconciler  *RedisUserReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-user-%d", time.Now().UnixNano())
			clusterName = fmt.Sprintf("test-cluster-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			// Create parent RedisCluster
			cluster = &cachev1beta1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: namespace,
				},
				Spec: cachev1beta1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1beta1.StorageSpec{
						Size: "10Gi",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			cr = &cachev1beta1.RedisUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1beta1.RedisUserSpec{
					Username:   "appuser",
					ClusterRef: clusterName,
				},
			}

			reconciler = &RedisUserReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			// Re-fetch to get UID for owner references
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &cachev1beta1.RedisUser{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
			clusterRes := &cachev1beta1.RedisCluster{}
			clusterKey := types.NamespacedName{Name: clusterName, Namespace: namespace}
			if err := k8sClient.Get(ctx, clusterKey, clusterRes); err == nil {
				clusterRes.Finalizers = nil
				_ = k8sClient.Update(ctx, clusterRes)
				Expect(k8sClient.Delete(ctx, clusterRes)).To(Succeed())
			}
		})

		It("should create Secret with REDIS_USER_PASSWORD key when absent", func() {
			Expect(reconciler.reconcileUserSecret(ctx, cr)).To(Succeed())

			secret := &corev1.Secret{}
			secretKey := types.NamespacedName{Name: fmt.Sprintf("%s-user-secret", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, secretKey, secret)).To(Succeed())

			// Verify credential key exists
			Expect(secret.Data).To(HaveKey("REDIS_USER_PASSWORD"))

			// Verify owner reference
			Expect(secret.OwnerReferences).To(HaveLen(1))
			Expect(secret.OwnerReferences[0].Name).To(Equal(name))

			// Verify labels
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "redis-user"))
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", name))
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "redis-operator"))
			Expect(secret.Labels).To(HaveKeyWithValue("app.kubernetes.io/component", "user"))
		})

		It("should not recreate existing Secret (idempotent)", func() {
			Expect(reconciler.reconcileUserSecret(ctx, cr)).To(Succeed())

			secret := &corev1.Secret{}
			secretKey := types.NamespacedName{Name: fmt.Sprintf("%s-user-secret", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, secretKey, secret)).To(Succeed())
			originalVersion := secret.ResourceVersion

			// Reconcile again
			Expect(reconciler.reconcileUserSecret(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, secretKey, secret)).To(Succeed())
			Expect(secret.ResourceVersion).To(Equal(originalVersion))
		})

		It("should skip Secret creation when passwordSecret is provided", func() {
			// Create an external secret that the user references
			externalSecretName := fmt.Sprintf("%s-external-pw", name)
			externalSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      externalSecretName,
					Namespace: namespace,
				},
				StringData: map[string]string{
					"REDIS_USER_PASSWORD": "externally-managed-password",
				},
			}
			Expect(k8sClient.Create(ctx, externalSecret)).To(Succeed())

			// Set passwordSecret on the CR
			cr.Spec.PasswordSecret = externalSecretName
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())

			Expect(reconciler.reconcileUserSecret(ctx, cr)).To(Succeed())

			// Verify that the operator-managed secret was NOT created
			operatorSecret := &corev1.Secret{}
			operatorSecretKey := types.NamespacedName{Name: fmt.Sprintf("%s-user-secret", name), Namespace: namespace}
			err := k8sClient.Get(ctx, operatorSecretKey, operatorSecret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))

			// Verify status references the external secret
			Expect(cr.Status.PasswordSecretName).To(Equal(externalSecretName))

			// Cleanup external secret
			Expect(k8sClient.Delete(ctx, externalSecret)).To(Succeed())
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileUserACL
	// ============================================================
	Context("When reconciling UserACL", func() {
		var (
			ctx         context.Context
			name        string
			clusterName string
			namespace   string
			key         types.NamespacedName
			cr          *cachev1beta1.RedisUser
			cluster     *cachev1beta1.RedisCluster
			reconciler  *RedisUserReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-user-%d", time.Now().UnixNano())
			clusterName = fmt.Sprintf("test-cluster-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			// Create parent RedisCluster
			cluster = &cachev1beta1.RedisCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: namespace,
				},
				Spec: cachev1beta1.RedisClusterSpec{
					Replicas: 3,
					Version:  "7.4",
					Storage: cachev1beta1.StorageSpec{
						Size: "10Gi",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			cr = &cachev1beta1.RedisUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: cachev1beta1.RedisUserSpec{
					Username:    "appuser",
					ClusterRef:  clusterName,
					Permissions: []string{"+@read", "~app:*"},
				},
			}

			reconciler = &RedisUserReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &cachev1beta1.RedisUser{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
			clusterRes := &cachev1beta1.RedisCluster{}
			clusterKey := types.NamespacedName{Name: clusterName, Namespace: namespace}
			if err := k8sClient.Get(ctx, clusterKey, clusterRes); err == nil {
				clusterRes.Finalizers = nil
				_ = k8sClient.Update(ctx, clusterRes)
				Expect(k8sClient.Delete(ctx, clusterRes)).To(Succeed())
			}
		})

		It("should create ConfigMap with users.acl when absent", func() {
			Expect(reconciler.reconcileUserACL(ctx, cr)).To(Succeed())

			configMap := &corev1.ConfigMap{}
			cmKey := types.NamespacedName{Name: fmt.Sprintf("%s-acl", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())

			// Verify ACL content
			Expect(configMap.Data).To(HaveKey("users.acl"))
			Expect(configMap.Data["users.acl"]).To(ContainSubstring("user appuser on"))
			Expect(configMap.Data["users.acl"]).To(ContainSubstring("+@read"))
			Expect(configMap.Data["users.acl"]).To(ContainSubstring("~app:*"))

			// Verify owner reference
			Expect(configMap.OwnerReferences).To(HaveLen(1))
			Expect(configMap.OwnerReferences[0].Name).To(Equal(name))

			// Verify labels
			Expect(configMap.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "redis-user"))
			Expect(configMap.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "redis-operator"))
			Expect(configMap.Labels).To(HaveKeyWithValue("app.kubernetes.io/component", "user"))
		})

		It("should not recreate existing ACL ConfigMap (idempotent)", func() {
			Expect(reconciler.reconcileUserACL(ctx, cr)).To(Succeed())

			configMap := &corev1.ConfigMap{}
			cmKey := types.NamespacedName{Name: fmt.Sprintf("%s-acl", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())
			originalVersion := configMap.ResourceVersion

			// Reconcile again
			Expect(reconciler.reconcileUserACL(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())
			Expect(configMap.ResourceVersion).To(Equal(originalVersion))
		})

		It("should update ACL ConfigMap when permissions change", func() {
			Expect(reconciler.reconcileUserACL(ctx, cr)).To(Succeed())

			configMap := &corev1.ConfigMap{}
			cmKey := types.NamespacedName{Name: fmt.Sprintf("%s-acl", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())
			originalVersion := configMap.ResourceVersion
			Expect(configMap.Data["users.acl"]).To(ContainSubstring("+@read"))

			// Change permissions
			cr.Spec.Permissions = []string{"+@all", "~*"}
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())

			Expect(reconciler.reconcileUserACL(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, cmKey, configMap)).To(Succeed())

			// Verify ConfigMap was updated (ResourceVersion changed)
			Expect(configMap.ResourceVersion).NotTo(Equal(originalVersion))
			Expect(configMap.Data["users.acl"]).To(ContainSubstring("+@all"))
			Expect(configMap.Data["users.acl"]).To(ContainSubstring("~*"))
		})
	})

	// ============================================================
	// Helper Function Tests
	// ============================================================
	Context("When testing helper functions", func() {
		It("should return correct labels for RedisUser", func() {
			cr := &cachev1beta1.RedisUser{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-user",
				},
				Spec: cachev1beta1.RedisUserSpec{
					Username:   "testuser",
					ClusterRef: "my-cluster",
				},
			}

			labels := labelsForRedisUser(cr)

			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/name", "redis-user"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/instance", "my-user"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "redis-operator"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/part-of", "my-cluster"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/component", "user"))
			Expect(labels).To(HaveLen(5))
		})

		It("should build ACL config with explicit permissions", func() {
			cr := &cachev1beta1.RedisUser{
				Spec: cachev1beta1.RedisUserSpec{
					Username:    "readuser",
					Permissions: []string{"+@read", "~cache:*"},
				},
			}

			acl := buildACLConfig(cr)

			Expect(acl).To(Equal("user readuser on +@read ~cache:*\n"))
		})

		It("should build ACL config with default permissions when none specified", func() {
			cr := &cachev1beta1.RedisUser{
				Spec: cachev1beta1.RedisUserSpec{
					Username: "defaultuser",
				},
			}

			acl := buildACLConfig(cr)

			Expect(acl).To(Equal("user defaultuser on ~* +@all\n"))
		})
	})
})
