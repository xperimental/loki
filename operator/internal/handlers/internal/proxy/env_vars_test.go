package proxy

import (
	"testing"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/manifests"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestGetEnvVars_ReturnsNil_WhenNoClusterProxySet(t *testing.T) {
	require.Nil(t, GetEnvVars(nil))
}

func TestGetEnvVars_ReturnsNil_WhenClusterProxyReadFromEnvVarsFalse(t *testing.T) {
	require.Nil(t, GetEnvVars(&lokiv1.ClusterProxy{ReadVarsFromEnv: false}))
}

func TestGetEnvVars_ReturnEnvVars(t *testing.T) {
	expected := []corev1.EnvVar{}
	for _, key := range manifests.ProxyEnvNames {
		t.Setenv(key, "test")
		expected = append(expected, corev1.EnvVar{Name: key, Value: "test"})
	}

	actual := GetEnvVars(&lokiv1.ClusterProxy{ReadVarsFromEnv: true})
	require.ElementsMatch(t, actual, expected)
}
