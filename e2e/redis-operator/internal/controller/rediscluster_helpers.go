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
	"fmt"
	"math/big"
	"strings"

	cachev1beta1 "github.com/example/redis-operator/api/v1beta1"
)

const (
	passwordLength  = 24
	passwordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// labelsForRedisCluster returns the standard labels for all resources managed by the operator.
func labelsForRedisCluster(cr *cachev1beta1.RedisCluster) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "redis",
		"app.kubernetes.io/instance":   cr.Name,
		"app.kubernetes.io/managed-by": "redis-operator",
		"app.kubernetes.io/part-of":    cr.Name,
		"app.kubernetes.io/version":    cr.Spec.Version,
	}
}

// imageForRedisCluster returns the container image for the given Redis version.
func imageForRedisCluster(cr *cachev1beta1.RedisCluster) string {
	majorVersion := strings.Split(cr.Spec.Version, ".")[0]
	return fmt.Sprintf("registry.redhat.io/rhel9/redis-%s", majorVersion)
}

// generatePassword creates a cryptographically random password.
func generatePassword() string {
	result := make([]byte, passwordLength)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordCharset))))
		if err != nil {
			// Fallback should never happen with crypto/rand, but be safe
			result[i] = passwordCharset[0]
			continue
		}
		result[i] = passwordCharset[n.Int64()]
	}
	return string(result)
}
