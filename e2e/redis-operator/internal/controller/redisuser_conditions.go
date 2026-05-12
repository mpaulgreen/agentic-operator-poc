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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1beta1 "github.com/example/redis-operator/api/v1beta1"
)

const (
	RedisUserConditionAvailable = "Available"
	RedisUserConditionDegraded  = "Degraded"
)

func setRedisUserCondition(cr *cachev1beta1.RedisUser, conditionType string, status metav1.ConditionStatus, reason, message string) {
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
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: cr.Generation,
	})
}

func setRedisUserAvailableCondition(cr *cachev1beta1.RedisUser, reason, message string) {
	setRedisUserCondition(cr, RedisUserConditionAvailable, metav1.ConditionTrue, reason, message)
}

func setRedisUserUnavailableCondition(cr *cachev1beta1.RedisUser, reason, message string) {
	setRedisUserCondition(cr, RedisUserConditionAvailable, metav1.ConditionFalse, reason, message)
}

func setRedisUserDegradedCondition(cr *cachev1beta1.RedisUser, reason, message string) {
	setRedisUserCondition(cr, RedisUserConditionDegraded, metav1.ConditionTrue, reason, message)
}

func clearRedisUserDegradedCondition(cr *cachev1beta1.RedisUser, reason, message string) {
	setRedisUserCondition(cr, RedisUserConditionDegraded, metav1.ConditionFalse, reason, message)
}
