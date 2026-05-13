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
	backupv1beta1 "github.com/example/mongodb-operator/api/backup/v1beta1"
)

func labelsForBackupPolicy(cr *backupv1beta1.MongoBackupPolicy) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "mongodb-backup",
		"app.kubernetes.io/instance":   cr.Name,
		"app.kubernetes.io/managed-by": "mongodb-operator",
		"app.kubernetes.io/part-of":    cr.Spec.ClusterRef,
		"app.kubernetes.io/component":  "backup-policy",
	}
}
