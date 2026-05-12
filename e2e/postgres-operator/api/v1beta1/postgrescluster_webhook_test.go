package v1beta1

import "testing"

func TestValidate_MaxClientConnectionsLessThanPoolSize(t *testing.T) {
	cr := &PostgresCluster{
		Spec: PostgresClusterSpec{
			Replicas: 3, Version: "16",
			Storage:        StorageSpec{Size: "1Gi"},
			ConnectionPool: &ConnectionPoolSpec{Enabled: true, PoolSize: 50, MaxClientConnections: 10},
		},
	}
	_, err := cr.ValidateCreate()
	if err == nil {
		t.Error("expected error for maxClientConnections < poolSize")
	}
}

func TestDefault_ConnectionPoolDefaults(t *testing.T) {
	cr := &PostgresCluster{
		Spec: PostgresClusterSpec{
			Replicas: 3, Version: "16",
			Storage:        StorageSpec{Size: "1Gi"},
			ConnectionPool: &ConnectionPoolSpec{Enabled: true},
		},
	}
	cr.Default()
	if cr.Spec.ConnectionPool.PoolSize != 10 {
		t.Errorf("expected poolSize=10, got %d", cr.Spec.ConnectionPool.PoolSize)
	}
	if cr.Spec.ConnectionPool.MaxClientConnections != 100 {
		t.Errorf("expected maxClientConnections=100, got %d", cr.Spec.ConnectionPool.MaxClientConnections)
	}
	if cr.Spec.ConnectionPool.IdleTimeout != "30s" {
		t.Errorf("expected idleTimeout=30s, got %s", cr.Spec.ConnectionPool.IdleTimeout)
	}
}
