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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1beta1 "github.com/example/mongodb-operator/api/backup/v1beta1"
	databasev1beta1 "github.com/example/mongodb-operator/api/v1beta1"
)

var _ = Describe("MongoBackupPolicy Controller", func() {

	// ============================================================
	// Lifecycle Tests
	// ============================================================
	Context("When reconciling a MongoBackupPolicy", func() {
		var (
			ctx         context.Context
			name        string
			clusterName string
			namespace   string
			key         types.NamespacedName
			cr          *backupv1beta1.MongoBackupPolicy
			cluster     *databasev1beta1.MongoCluster
			reconciler  *MongoBackupPolicyReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			clusterName = fmt.Sprintf("cluster-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			// Create the parent MongoCluster that clusterRef points to
			cluster = &databasev1beta1.MongoCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: namespace,
				},
				Spec: databasev1beta1.MongoClusterSpec{
					Replicas: 3,
					Version:  "7.0",
					Storage: databasev1beta1.StorageSpec{
						Size: "10Gi",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			cr = &backupv1beta1.MongoBackupPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: backupv1beta1.MongoBackupPolicySpec{
					ClusterRef:    clusterName,
					Schedule:      "0 2 * * *",
					RetentionDays: 30,
					StorageSize:   "5Gi",
				},
			}

			reconciler = &MongoBackupPolicyReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &backupv1beta1.MongoBackupPolicy{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
			// Clean up parent cluster
			clusterRes := &databasev1beta1.MongoCluster{}
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

			updated := &backupv1beta1.MongoBackupPolicy{}
			Expect(k8sClient.Get(ctx, key, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement("backup.mongodb.example.com/finalizer"))
		})

		It("should create all managed resources", func() {
			// Multiple reconciliations to ensure all resources are created
			for i := 0; i < 3; i++ {
				_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			}

			// Verify CronJob
			cronJob := &batchv1.CronJob{}
			cronJobKey := types.NamespacedName{Name: fmt.Sprintf("%s-cronjob", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cronJobKey, cronJob)).To(Succeed())

			// Verify PVC
			pvc := &corev1.PersistentVolumeClaim{}
			pvcKey := types.NamespacedName{Name: fmt.Sprintf("%s-storage", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, pvcKey, pvc)).To(Succeed())
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
			updated := &backupv1beta1.MongoBackupPolicy{}
			Expect(k8sClient.Get(ctx, key, updated)).To(Succeed())
			Expect(updated.Finalizers).NotTo(BeEmpty())

			// Delete the resource
			Expect(k8sClient.Delete(ctx, updated)).To(Succeed())

			// Reconcile should handle deletion and remove finalizer
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was removed (resource may or may not still exist)
			deleted := &backupv1beta1.MongoBackupPolicy{}
			err = k8sClient.Get(ctx, key, deleted)
			if err == nil {
				Expect(deleted.Finalizers).To(BeEmpty())
			}
			// If err != nil, resource was already garbage collected -- expected
		})
	})

	// ============================================================
	// Per-Method Tests: reconcilePolicyCronJob
	// ============================================================
	Context("When reconciling Policy CronJob", func() {
		var (
			ctx         context.Context
			name        string
			clusterName string
			namespace   string
			key         types.NamespacedName
			cr          *backupv1beta1.MongoBackupPolicy
			cluster     *databasev1beta1.MongoCluster
			reconciler  *MongoBackupPolicyReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			clusterName = fmt.Sprintf("cluster-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cluster = &databasev1beta1.MongoCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: namespace,
				},
				Spec: databasev1beta1.MongoClusterSpec{
					Replicas: 3,
					Version:  "7.0",
					Storage: databasev1beta1.StorageSpec{
						Size: "10Gi",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			cr = &backupv1beta1.MongoBackupPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: backupv1beta1.MongoBackupPolicySpec{
					ClusterRef:    clusterName,
					Schedule:      "0 2 * * *",
					RetentionDays: 30,
					StorageSize:   "5Gi",
				},
			}

			reconciler = &MongoBackupPolicyReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &backupv1beta1.MongoBackupPolicy{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
			clusterRes := &databasev1beta1.MongoCluster{}
			clusterKey := types.NamespacedName{Name: clusterName, Namespace: namespace}
			if err := k8sClient.Get(ctx, clusterKey, clusterRes); err == nil {
				clusterRes.Finalizers = nil
				_ = k8sClient.Update(ctx, clusterRes)
				Expect(k8sClient.Delete(ctx, clusterRes)).To(Succeed())
			}
		})

		It("should create CronJob with correct schedule when absent", func() {
			Expect(reconciler.reconcilePolicyCronJob(ctx, cr)).To(Succeed())

			cronJob := &batchv1.CronJob{}
			cronJobKey := types.NamespacedName{Name: fmt.Sprintf("%s-cronjob", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cronJobKey, cronJob)).To(Succeed())

			// Verify schedule matches spec
			Expect(cronJob.Spec.Schedule).To(Equal("0 2 * * *"))

			// Verify container
			Expect(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Name).To(Equal("backup"))

			// Verify restart policy
			Expect(cronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyOnFailure))

			// Verify owner reference
			Expect(cronJob.OwnerReferences).To(HaveLen(1))
			Expect(cronJob.OwnerReferences[0].Name).To(Equal(name))

			// Verify labels
			Expect(cronJob.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "mongodb-backup"))
			Expect(cronJob.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", name))
			Expect(cronJob.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "mongodb-operator"))
		})

		It("should not recreate existing CronJob (idempotent)", func() {
			Expect(reconciler.reconcilePolicyCronJob(ctx, cr)).To(Succeed())

			cronJob := &batchv1.CronJob{}
			cronJobKey := types.NamespacedName{Name: fmt.Sprintf("%s-cronjob", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, cronJobKey, cronJob)).To(Succeed())
			originalVersion := cronJob.ResourceVersion

			// Reconcile again
			Expect(reconciler.reconcilePolicyCronJob(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, cronJobKey, cronJob)).To(Succeed())
			Expect(cronJob.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Per-Method Tests: reconcileBackupPVC
	// ============================================================
	Context("When reconciling Backup PVC", func() {
		var (
			ctx         context.Context
			name        string
			clusterName string
			namespace   string
			key         types.NamespacedName
			cr          *backupv1beta1.MongoBackupPolicy
			cluster     *databasev1beta1.MongoCluster
			reconciler  *MongoBackupPolicyReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			clusterName = fmt.Sprintf("cluster-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cluster = &databasev1beta1.MongoCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: namespace,
				},
				Spec: databasev1beta1.MongoClusterSpec{
					Replicas: 3,
					Version:  "7.0",
					Storage: databasev1beta1.StorageSpec{
						Size: "10Gi",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			cr = &backupv1beta1.MongoBackupPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: backupv1beta1.MongoBackupPolicySpec{
					ClusterRef:    clusterName,
					Schedule:      "0 2 * * *",
					RetentionDays: 30,
					StorageSize:   "5Gi",
				},
			}

			reconciler = &MongoBackupPolicyReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &backupv1beta1.MongoBackupPolicy{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
			clusterRes := &databasev1beta1.MongoCluster{}
			clusterKey := types.NamespacedName{Name: clusterName, Namespace: namespace}
			if err := k8sClient.Get(ctx, clusterKey, clusterRes); err == nil {
				clusterRes.Finalizers = nil
				_ = k8sClient.Update(ctx, clusterRes)
				Expect(k8sClient.Delete(ctx, clusterRes)).To(Succeed())
			}
		})

		It("should create PVC with correct storage size when absent", func() {
			Expect(reconciler.reconcileBackupPVC(ctx, cr)).To(Succeed())

			pvc := &corev1.PersistentVolumeClaim{}
			pvcKey := types.NamespacedName{Name: fmt.Sprintf("%s-storage", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, pvcKey, pvc)).To(Succeed())

			// Verify storage size
			storageReq := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
			Expect(storageReq.String()).To(Equal("5Gi"))

			// Verify access mode
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))

			// Verify owner reference
			Expect(pvc.OwnerReferences).To(HaveLen(1))
			Expect(pvc.OwnerReferences[0].Name).To(Equal(name))

			// Verify labels
			Expect(pvc.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "mongodb-backup"))
			Expect(pvc.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", name))
			Expect(pvc.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "mongodb-operator"))
		})

		It("should not recreate existing PVC (idempotent)", func() {
			Expect(reconciler.reconcileBackupPVC(ctx, cr)).To(Succeed())

			pvc := &corev1.PersistentVolumeClaim{}
			pvcKey := types.NamespacedName{Name: fmt.Sprintf("%s-storage", name), Namespace: namespace}
			Expect(k8sClient.Get(ctx, pvcKey, pvc)).To(Succeed())
			originalVersion := pvc.ResourceVersion

			// Reconcile again
			Expect(reconciler.reconcileBackupPVC(ctx, cr)).To(Succeed())
			Expect(k8sClient.Get(ctx, pvcKey, pvc)).To(Succeed())
			Expect(pvc.ResourceVersion).To(Equal(originalVersion))
		})
	})

	// ============================================================
	// Parent Reference Tests
	// ============================================================
	Context("When referencing a parent MongoCluster", func() {
		var (
			ctx        context.Context
			name       string
			namespace  string
			key        types.NamespacedName
			cr         *backupv1beta1.MongoBackupPolicy
			reconciler *MongoBackupPolicyReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			name = fmt.Sprintf("test-%d", time.Now().UnixNano())
			namespace = "default"
			key = types.NamespacedName{Name: name, Namespace: namespace}

			cr = &backupv1beta1.MongoBackupPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: backupv1beta1.MongoBackupPolicySpec{
					ClusterRef:    "non-existent-cluster",
					Schedule:      "0 2 * * *",
					RetentionDays: 30,
					StorageSize:   "5Gi",
				},
			}

			reconciler = &MongoBackupPolicyReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(100),
			}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		})

		AfterEach(func() {
			resource := &backupv1beta1.MongoBackupPolicy{}
			if err := k8sClient.Get(ctx, key, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should reject non-existent clusterRef", func() {
			// First reconcile adds finalizer and sets Pending status
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})

			// Second reconcile hits the cluster lookup (after finalizer is added)
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))

			// Verify status is set to Failed
			updated := &backupv1beta1.MongoBackupPolicy{}
			Expect(k8sClient.Get(ctx, key, updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal("Failed"))
		})
	})

	// ============================================================
	// Helper Function Tests
	// ============================================================
	Context("When testing helper functions", func() {
		It("should return correct labels with expected keys and values", func() {
			cr := &backupv1beta1.MongoBackupPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-backup-policy",
				},
				Spec: backupv1beta1.MongoBackupPolicySpec{
					ClusterRef: "my-cluster",
				},
			}

			labels := labelsForBackupPolicy(cr)

			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/name", "mongodb-backup"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/instance", "my-backup-policy"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "mongodb-operator"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/part-of", "my-cluster"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/component", "backup-policy"))
			Expect(labels).To(HaveLen(5))
		})
	})
})
