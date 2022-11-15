package openshift

import (
	"context"
	"testing"

	"github.com/grafana/loki/operator/internal/external/k8s/k8sfakes"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetProxyEnvVars_ReturnError_WhenOtherThanNotFound(t *testing.T) {
	k := &k8sfakes.FakeClient{}

	k.GetStub = func(_ context.Context, name types.NamespacedName, object client.Object, _ ...client.GetOption) error {
		return apierrors.NewBadRequest("bad request")
	}

	_, err := GetProxyEnvVars(context.TODO(), k)
	require.Error(t, err)
}

func TestGetProxyEnvVars_ReturnEmpty_WhenNotFound(t *testing.T) {
	k := &k8sfakes.FakeClient{}

	k.GetStub = func(_ context.Context, name types.NamespacedName, object client.Object, _ ...client.GetOption) error {
		return apierrors.NewNotFound(schema.GroupResource{}, "something wasn't found")
	}

	envVars, err := GetProxyEnvVars(context.TODO(), k)
	require.NoError(t, err)
	require.Nil(t, envVars)
}

func TestGetProxyEnvVars_ReturnEnvVars_WhenProxyExists(t *testing.T) {
	k := &k8sfakes.FakeClient{}

	k.GetStub = func(_ context.Context, name types.NamespacedName, out client.Object, _ ...client.GetOption) error {
		if name.Name == proxyName {
			k.SetClientObject(out, &configv1.Proxy{
				Spec: configv1.ProxySpec{
					HTTPProxy:  "http-test",
					HTTPSProxy: "https-test",
					NoProxy:    "noproxy-test",
				},
			})
			return nil
		}
		return apierrors.NewNotFound(schema.GroupResource{}, "something wasn't found")
	}

	envVars, err := GetProxyEnvVars(context.TODO(), k)
	require.NoError(t, err)
	require.Contains(t, envVars, corev1.EnvVar{Name: "HTTP_PROXY", Value: "http-test"})
	require.Contains(t, envVars, corev1.EnvVar{Name: "http_proxy", Value: "http-test"})
	require.Contains(t, envVars, corev1.EnvVar{Name: "HTTPS_PROXY", Value: "https-test"})
	require.Contains(t, envVars, corev1.EnvVar{Name: "https_proxy", Value: "https-test"})
	require.Contains(t, envVars, corev1.EnvVar{Name: "NO_PROXY", Value: "noproxy-test"})
	require.Contains(t, envVars, corev1.EnvVar{Name: "no_proxy", Value: "noproxy-test"})
}
