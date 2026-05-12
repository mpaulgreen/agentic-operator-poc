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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cachev1beta1 "github.com/example/redis-operator/api/v1beta1"
)

func (r *RedisUserReconciler) reconcileUserSecret(ctx context.Context, cr *cachev1beta1.RedisUser) error {
	if cr.Spec.PasswordSecret != "" {
		cr.Status.PasswordSecretName = cr.Spec.PasswordSecret
		return nil
	}

	name := fmt.Sprintf("%s-user-secret", cr.Name)
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		cr.Status.PasswordSecretName = name
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForRedisUser(cr),
		},
		StringData: map[string]string{
			"REDIS_USER_PASSWORD": generatePassword(),
		},
	}

	if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, secret); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "SecretFailed",
			fmt.Sprintf("Failed to create user Secret: %v", err))
		return err
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "SecretCreated",
		fmt.Sprintf("Created user Secret %s", name))
	cr.Status.PasswordSecretName = name
	return nil
}

func (r *RedisUserReconciler) reconcileUserACL(ctx context.Context, cr *cachev1beta1.RedisUser) error {
	name := fmt.Sprintf("%s-acl", cr.Name)
	desiredACL := buildACLConfig(cr)

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cr.Namespace}, existing)
	if err == nil {
		if existing.Data["users.acl"] != desiredACL {
			existing.Data["users.acl"] = desiredACL
			if err := r.Update(ctx, existing); err != nil {
				r.Recorder.Event(cr, corev1.EventTypeWarning, "ACLConfigMapUpdateFailed",
					fmt.Sprintf("Failed to update ACL ConfigMap: %v", err))
				return err
			}
			r.Recorder.Event(cr, corev1.EventTypeNormal, "ACLConfigMapUpdated",
				fmt.Sprintf("Updated ACL ConfigMap %s", name))
		}
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForRedisUser(cr),
		},
		Data: map[string]string{
			"users.acl": desiredACL,
		},
	}

	if err := controllerutil.SetControllerReference(cr, configMap, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, configMap); err != nil {
		r.Recorder.Event(cr, corev1.EventTypeWarning, "ACLConfigMapFailed",
			fmt.Sprintf("Failed to create ACL ConfigMap: %v", err))
		return err
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "ACLConfigMapCreated",
		fmt.Sprintf("Created ACL ConfigMap %s", name))
	return nil
}
