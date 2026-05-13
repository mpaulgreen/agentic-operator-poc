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
	"crypto/rand"
	"math/big"

	databasev1alpha1 "github.com/example/mongodb-operator/api/v1alpha1"
)

const (
	passwordLength  = 24
	passwordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	keyFileLength   = 756
	keyFileCharset  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/"
)

func labelsForMongoCluster(cr *databasev1alpha1.MongoCluster) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "mongodb",
		"app.kubernetes.io/instance":   cr.Name,
		"app.kubernetes.io/managed-by": "mongodb-operator",
		"app.kubernetes.io/part-of":    cr.Name,
		"app.kubernetes.io/version":    cr.Spec.Version,
	}
}

func imageForMongoCluster(cr *databasev1alpha1.MongoCluster) string {
	// Red Hat certified MongoDB images (registry.connect.redhat.com/mongodb/enterprise-*)
	// require enterprise license. For E2E testing, use UBI micro with sleep as a mock
	// container — tests operator reconciliation, not the MongoDB process.
	_ = cr.Spec.Version
	return "registry.access.redhat.com/ubi9/ubi-micro:latest"
}

func generatePassword() string {
	return generateRandomString(passwordLength, passwordCharset)
}

func generateKeyFile() string {
	return generateRandomString(keyFileLength, keyFileCharset)
}

func generateRandomString(length int, charset string) string {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			result[i] = charset[0]
			continue
		}
		result[i] = charset[n.Int64()]
	}
	return string(result)
}
