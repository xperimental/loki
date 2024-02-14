package manifests

import (
	"fmt"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	configv1 "github.com/grafana/loki/operator/apis/config/v1"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
)

// Test that all serviceMonitor match the labels of their services so that we know all serviceMonitor
// will work when deployed.
func TestServiceMonitorMatchLabels(t *testing.T) {
	type test struct {
		Service    *corev1.Service
		PodMonitor *monitoringv1.PodMonitor
	}

	featureGates := configv1.FeatureGates{
		BuiltInCertManagement:      configv1.BuiltInCertManagement{Enabled: true},
		ServiceMonitors:            true,
		ServiceMonitorTLSEndpoints: true,
	}

	opt := Options{
		Name:      "test",
		Namespace: "test",
		Image:     "test",
		Gates:     featureGates,
		Stack: lokiv1.LokiStackSpec{
			Size: lokiv1.SizeOneXExtraSmall,
			Template: &lokiv1.LokiTemplateSpec{
				Compactor: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Distributor: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Ingester: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Querier: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				QueryFrontend: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Gateway: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				IndexGateway: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Ruler: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
			},
		},
	}

	table := []test{
		{
			Service:    NewDistributorHTTPService(opt),
			PodMonitor: NewDistributorPodMonitor(opt),
		},
		{
			Service:    NewIngesterHTTPService(opt),
			PodMonitor: NewIngesterPodMonitor(opt),
		},
		{
			Service:    NewQuerierHTTPService(opt),
			PodMonitor: NewQuerierPodMonitor(opt),
		},
		{
			Service:    NewQueryFrontendHTTPService(opt),
			PodMonitor: NewQueryFrontendPodMonitor(opt),
		},
		{
			Service:    NewCompactorHTTPService(opt),
			PodMonitor: NewCompactorPodMonitor(opt),
		},
		{
			Service:    NewGatewayHTTPService(opt),
			PodMonitor: NewGatewayPodMonitor(opt),
		},
		{
			Service:    NewIndexGatewayHTTPService(opt),
			PodMonitor: NewIndexGatewayPodMonitor(opt),
		},
		{
			Service:    NewRulerHTTPService(opt),
			PodMonitor: NewRulerPodMonitor(opt),
		},
	}

	for _, tst := range table {
		tst := tst
		testName := fmt.Sprintf("%s_%s", tst.Service.GetName(), tst.PodMonitor.GetName())
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			for k, v := range tst.PodMonitor.Spec.Selector.MatchLabels {
				if assert.Contains(t, tst.Service.Spec.Selector, k) {
					// only assert Equal if the previous assertion is successful or this will panic
					assert.Equal(t, v, tst.Service.Spec.Selector[k])
				}
			}
		})
	}
}

func TestServiceMonitorEndpoints_ForBuiltInCertRotation(t *testing.T) {
	type test struct {
		Service    *corev1.Service
		PodMonitor *monitoringv1.PodMonitor
	}

	featureGates := configv1.FeatureGates{
		BuiltInCertManagement:      configv1.BuiltInCertManagement{Enabled: true},
		ServiceMonitors:            true,
		ServiceMonitorTLSEndpoints: true,
	}

	opt := Options{
		Name:      "test",
		Namespace: "test",
		Image:     "test",
		Gates:     featureGates,
		Stack: lokiv1.LokiStackSpec{
			Size: lokiv1.SizeOneXExtraSmall,
			Template: &lokiv1.LokiTemplateSpec{
				Compactor: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Distributor: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Ingester: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Querier: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				QueryFrontend: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				IndexGateway: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
				Ruler: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
			},
		},
	}

	table := []test{
		{
			Service:    NewDistributorHTTPService(opt),
			PodMonitor: NewDistributorPodMonitor(opt),
		},
		{
			Service:    NewIngesterHTTPService(opt),
			PodMonitor: NewIngesterPodMonitor(opt),
		},
		{
			Service:    NewQuerierHTTPService(opt),
			PodMonitor: NewQuerierPodMonitor(opt),
		},
		{
			Service:    NewQueryFrontendHTTPService(opt),
			PodMonitor: NewQueryFrontendPodMonitor(opt),
		},
		{
			Service:    NewCompactorHTTPService(opt),
			PodMonitor: NewCompactorPodMonitor(opt),
		},
		{
			Service:    NewIndexGatewayHTTPService(opt),
			PodMonitor: NewIndexGatewayPodMonitor(opt),
		},
		{
			Service:    NewRulerHTTPService(opt),
			PodMonitor: NewRulerPodMonitor(opt),
		},
	}

	for _, tst := range table {
		tst := tst
		testName := fmt.Sprintf("%s_%s", tst.Service.GetName(), tst.PodMonitor.GetName())
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tst.PodMonitor.Spec.PodMetricsEndpoints)
			require.NotNil(t, tst.PodMonitor.Spec.PodMetricsEndpoints[0].TLSConfig)

			// Do not use bearer authentication for loki endpoints
			require.Empty(t, tst.PodMonitor.Spec.PodMetricsEndpoints[0].BearerTokenSecret)

			// Check using built-in PKI
			c := tst.PodMonitor.Spec.PodMetricsEndpoints[0].TLSConfig
			require.Equal(t, c.CA.ConfigMap.LocalObjectReference.Name, signingCABundleName(opt.Name))
			require.Equal(t, c.Cert.Secret.LocalObjectReference.Name, tst.Service.Name)
			require.Equal(t, c.KeySecret.LocalObjectReference.Name, tst.Service.Name)
		})
	}
}

func TestServiceMonitorEndpoints_ForGatewayServiceMonitor(t *testing.T) {
	tt := []struct {
		desc  string
		opts  Options
		total int
		want  []monitoringv1.PodMetricsEndpoint
	}{
		{
			desc: "default",
			opts: Options{
				Name:      "test",
				Namespace: "test",
				Image:     "test",
				Stack: lokiv1.LokiStackSpec{
					Size: lokiv1.SizeOneXExtraSmall,
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.Static,
					},
					Template: &lokiv1.LokiTemplateSpec{
						Gateway: &lokiv1.LokiComponentSpec{
							Replicas: 1,
						},
					},
				},
			},
			total: 1,
			want: []monitoringv1.PodMetricsEndpoint{
				{
					Port:   gatewayInternalPortName,
					Path:   "/metrics",
					Scheme: "http",
				},
			},
		},
		{
			desc: "with http encryption",
			opts: Options{
				Name:      "test",
				Namespace: "test",
				Image:     "test",
				Gates: configv1.FeatureGates{
					LokiStackGateway:           true,
					BuiltInCertManagement:      configv1.BuiltInCertManagement{Enabled: true},
					ServiceMonitors:            true,
					ServiceMonitorTLSEndpoints: true,
				},
				Stack: lokiv1.LokiStackSpec{
					Size: lokiv1.SizeOneXExtraSmall,
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.Static,
					},
					Template: &lokiv1.LokiTemplateSpec{
						Gateway: &lokiv1.LokiComponentSpec{
							Replicas: 1,
						},
					},
				},
			},
			total: 1,
			want: []monitoringv1.PodMetricsEndpoint{
				{
					Port:   gatewayInternalPortName,
					Path:   "/metrics",
					Scheme: "https",
					BearerTokenSecret: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-gateway-token",
						},
						Key: corev1.ServiceAccountTokenKey,
					},
					TLSConfig: &monitoringv1.PodMetricsEndpointTLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							CA: monitoringv1.SecretOrConfigMap{
								ConfigMap: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: gatewaySigningCABundleName("test-gateway"),
									},
									Key: caFile,
								},
							},
							ServerName: "test-gateway-http.test.svc.cluster.local",
						},
					},
				},
			},
		},
		{
			desc: "openshift-logging",
			opts: Options{
				Name:      "test",
				Namespace: "test",
				Image:     "test",
				Stack: lokiv1.LokiStackSpec{
					Size: lokiv1.SizeOneXExtraSmall,
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.OpenshiftLogging,
					},
					Template: &lokiv1.LokiTemplateSpec{
						Gateway: &lokiv1.LokiComponentSpec{
							Replicas: 1,
						},
					},
				},
			},
			total: 2,
			want: []monitoringv1.PodMetricsEndpoint{
				{
					Port:   gatewayInternalPortName,
					Path:   "/metrics",
					Scheme: "http",
				},
				{
					Port:   "opa-metrics",
					Path:   "/metrics",
					Scheme: "http",
				},
			},
		},
		{
			desc: "openshift-logging with http encryption",
			opts: Options{
				Name:      "test",
				Namespace: "test",
				Image:     "test",
				Gates: configv1.FeatureGates{
					LokiStackGateway:           true,
					BuiltInCertManagement:      configv1.BuiltInCertManagement{Enabled: true},
					ServiceMonitors:            true,
					ServiceMonitorTLSEndpoints: true,
				},
				Stack: lokiv1.LokiStackSpec{
					Size: lokiv1.SizeOneXExtraSmall,
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.OpenshiftLogging,
					},
					Template: &lokiv1.LokiTemplateSpec{
						Gateway: &lokiv1.LokiComponentSpec{
							Replicas: 1,
						},
					},
				},
			},
			total: 2,
			want: []monitoringv1.PodMetricsEndpoint{
				{
					Port:   gatewayInternalPortName,
					Path:   "/metrics",
					Scheme: "https",
					BearerTokenSecret: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-gateway-token",
						},
						Key: corev1.ServiceAccountTokenKey,
					},
					TLSConfig: &monitoringv1.PodMetricsEndpointTLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							CA: monitoringv1.SecretOrConfigMap{
								ConfigMap: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: gatewaySigningCABundleName("test-gateway"),
									},
									Key: caFile,
								},
							},
							ServerName: "test-gateway-http.test.svc.cluster.local",
						},
					},
				},
				{
					Port:   "opa-metrics",
					Path:   "/metrics",
					Scheme: "https",
					BearerTokenSecret: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-gateway-token",
						},
						Key: corev1.ServiceAccountTokenKey,
					},
					TLSConfig: &monitoringv1.PodMetricsEndpointTLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							CA: monitoringv1.SecretOrConfigMap{
								ConfigMap: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: gatewaySigningCABundleName("test-gateway"),
									},
									Key: caFile,
								},
							},
							ServerName: "test-gateway-http.test.svc.cluster.local",
						},
					},
				},
			},
		},
		{
			desc: "openshift-network",
			opts: Options{
				Name:      "test",
				Namespace: "test",
				Image:     "test",
				Stack: lokiv1.LokiStackSpec{
					Size: lokiv1.SizeOneXExtraSmall,
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.OpenshiftNetwork,
					},
					Template: &lokiv1.LokiTemplateSpec{
						Gateway: &lokiv1.LokiComponentSpec{
							Replicas: 1,
						},
					},
				},
			},
			total: 2,
			want: []monitoringv1.PodMetricsEndpoint{
				{
					Port:   gatewayInternalPortName,
					Path:   "/metrics",
					Scheme: "http",
				},
				{
					Port:   "opa-metrics",
					Path:   "/metrics",
					Scheme: "http",
				},
			},
		},
		{
			desc: "openshift-network with http encryption",
			opts: Options{
				Name:      "test",
				Namespace: "test",
				Image:     "test",
				Gates: configv1.FeatureGates{
					LokiStackGateway:           true,
					BuiltInCertManagement:      configv1.BuiltInCertManagement{Enabled: true},
					ServiceMonitors:            true,
					ServiceMonitorTLSEndpoints: true,
				},
				Stack: lokiv1.LokiStackSpec{
					Size: lokiv1.SizeOneXExtraSmall,
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.OpenshiftNetwork,
					},
					Template: &lokiv1.LokiTemplateSpec{
						Gateway: &lokiv1.LokiComponentSpec{
							Replicas: 1,
						},
					},
				},
			},
			total: 2,
			want: []monitoringv1.PodMetricsEndpoint{
				{
					Port:   gatewayInternalPortName,
					Path:   "/metrics",
					Scheme: "https",
					BearerTokenSecret: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-gateway-token",
						},
						Key: corev1.ServiceAccountTokenKey,
					},
					TLSConfig: &monitoringv1.PodMetricsEndpointTLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							CA: monitoringv1.SecretOrConfigMap{
								ConfigMap: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: gatewaySigningCABundleName("test-gateway"),
									},
									Key: caFile,
								},
							},
							ServerName: "test-gateway-http.test.svc.cluster.local",
						},
					},
				},
				{
					Port:   "opa-metrics",
					Path:   "/metrics",
					Scheme: "https",
					BearerTokenSecret: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-gateway-token",
						},
						Key: corev1.ServiceAccountTokenKey,
					},
					TLSConfig: &monitoringv1.PodMetricsEndpointTLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							CA: monitoringv1.SecretOrConfigMap{
								ConfigMap: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: gatewaySigningCABundleName("test-gateway"),
									},
									Key: caFile,
								},
							},
							ServerName: "test-gateway-http.test.svc.cluster.local",
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			sm := NewGatewayPodMonitor(tc.opts)
			require.Len(t, sm.Spec.PodMetricsEndpoints, tc.total)

			for _, endpoint := range tc.want {
				require.Contains(t, sm.Spec.PodMetricsEndpoints, endpoint)
			}
		})
	}
}
