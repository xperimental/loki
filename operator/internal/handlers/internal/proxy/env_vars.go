package proxy

import (
	"os"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/manifests"
	corev1 "k8s.io/api/core/v1"
)

// GetEnvVars returns the six proxy environment variables if
// provided by the container environment, namely:
// - HTTP_PROXY and http_proxy
// - HTTPS_PROXY and https_proxy
// - NO_PROXY and no_proxy
func GetEnvVars(cp *lokiv1.ClusterProxy) []corev1.EnvVar {
	if cp == nil || !cp.ReadVarsFromEnv {
		return nil
	}

	var envVars []corev1.EnvVar
	for _, key := range manifests.ProxyEnvNames {
		value, nonEmpty := os.LookupEnv(key)
		if nonEmpty {
			envVars = append(envVars, corev1.EnvVar{
				Name:  key,
				Value: value,
			})
		}
	}

	return envVars
}
