package openshift

import (
	"context"

	"github.com/grafana/loki/operator/internal/external/k8s"
	"github.com/grafana/loki/operator/internal/manifests"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const proxyName = "cluster"

// GetProxyEnvVars returns the six proxy environment variables from configv1.Proxy
// if provided by the container environment, namely:
// - HTTP_PROXY and http_proxy
// - HTTPS_PROXY and https_proxy
// - NO_PROXY and no_proxy
func GetProxyEnvVars(ctx context.Context, k k8s.Client) ([]corev1.EnvVar, error) {
	key := client.ObjectKey{Name: proxyName}
	p := configv1.Proxy{}
	if err := k.Get(ctx, key, &p); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return manifests.ToEnvVars(p.Spec.HTTPProxy, p.Spec.HTTPSProxy, p.Spec.NoProxy), nil
}
