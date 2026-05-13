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

	backupv1beta1 "github.com/example/mongodb-operator/api/backup/v1beta1"
)

func (r *MongoBackupPolicyReconciler) updateBackupPolicyStatus(ctx context.Context, cr *backupv1beta1.MongoBackupPolicy) error {
	cr.Status.Phase = "Active"
	setAvailableCondition(cr, "ReconcileComplete", "All backup resources reconciled successfully")
	clearDegradedCondition(cr, "ReconcileComplete", "No errors")
	return r.Status().Update(ctx, cr)
}
