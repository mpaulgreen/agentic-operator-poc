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
	"fmt"
	"strings"

	cachev1beta1 "github.com/example/redis-operator/api/v1beta1"
)

func labelsForRedisUser(cr *cachev1beta1.RedisUser) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "redis-user",
		"app.kubernetes.io/instance":   cr.Name,
		"app.kubernetes.io/managed-by": "redis-operator",
		"app.kubernetes.io/part-of":    cr.Spec.ClusterRef,
		"app.kubernetes.io/component":  "user",
	}
}

func buildACLConfig(cr *cachev1beta1.RedisUser) string {
	acl := fmt.Sprintf("user %s on", cr.Spec.Username)
	if len(cr.Spec.Permissions) > 0 {
		acl += " " + strings.Join(cr.Spec.Permissions, " ")
	} else {
		acl += " ~* +@all"
	}
	return acl + "\n"
}
