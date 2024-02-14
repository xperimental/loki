package manifests

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildPodMonitors builds the service monitors
func BuildPodMonitors(opts Options) []client.Object {
	return []client.Object{
		NewDistributorPodMonitor(opts),
		NewIngesterPodMonitor(opts),
		NewQuerierPodMonitor(opts),
		NewCompactorPodMonitor(opts),
		NewQueryFrontendPodMonitor(opts),
		NewIndexGatewayPodMonitor(opts),
		NewRulerPodMonitor(opts),
		NewGatewayPodMonitor(opts),
	}
}

// NewDistributorPodMonitor creates a k8s service monitor for the distributor component
func NewDistributorPodMonitor(opts Options) *monitoringv1.PodMonitor {
	l := ComponentLabels(LabelDistributorComponent, opts.Name)

	serviceMonitorName := serviceMonitorName(DistributorName(opts.Name))
	serviceName := serviceNameDistributorHTTP(opts.Name)
	lokiEndpoint := lokiPodMetricsEndpoint(opts.Name, lokiHTTPPortName, serviceName, opts.Namespace, opts.Gates.ServiceMonitorTLSEndpoints)

	return newPodMonitor(opts.Namespace, serviceMonitorName, l, lokiEndpoint)
}

// NewIngesterPodMonitor creates a k8s service monitor for the ingester component
func NewIngesterPodMonitor(opts Options) *monitoringv1.PodMonitor {
	l := ComponentLabels(LabelIngesterComponent, opts.Name)

	serviceMonitorName := serviceMonitorName(IngesterName(opts.Name))
	serviceName := serviceNameIngesterHTTP(opts.Name)
	lokiEndpoint := lokiPodMetricsEndpoint(opts.Name, lokiHTTPPortName, serviceName, opts.Namespace, opts.Gates.ServiceMonitorTLSEndpoints)

	return newPodMonitor(opts.Namespace, serviceMonitorName, l, lokiEndpoint)
}

// NewQuerierPodMonitor creates a k8s service monitor for the querier component
func NewQuerierPodMonitor(opts Options) *monitoringv1.PodMonitor {
	l := ComponentLabels(LabelQuerierComponent, opts.Name)

	serviceMonitorName := serviceMonitorName(QuerierName(opts.Name))
	serviceName := serviceNameQuerierHTTP(opts.Name)
	lokiEndpoint := lokiPodMetricsEndpoint(opts.Name, lokiHTTPPortName, serviceName, opts.Namespace, opts.Gates.ServiceMonitorTLSEndpoints)

	return newPodMonitor(opts.Namespace, serviceMonitorName, l, lokiEndpoint)
}

// NewCompactorPodMonitor creates a k8s service monitor for the compactor component
func NewCompactorPodMonitor(opts Options) *monitoringv1.PodMonitor {
	l := ComponentLabels(LabelCompactorComponent, opts.Name)

	serviceMonitorName := serviceMonitorName(CompactorName(opts.Name))
	serviceName := serviceNameCompactorHTTP(opts.Name)
	lokiEndpoint := lokiPodMetricsEndpoint(opts.Name, lokiHTTPPortName, serviceName, opts.Namespace, opts.Gates.ServiceMonitorTLSEndpoints)

	return newPodMonitor(opts.Namespace, serviceMonitorName, l, lokiEndpoint)
}

// NewQueryFrontendPodMonitor creates a k8s service monitor for the query-frontend component
func NewQueryFrontendPodMonitor(opts Options) *monitoringv1.PodMonitor {
	l := ComponentLabels(LabelQueryFrontendComponent, opts.Name)

	serviceMonitorName := serviceMonitorName(QueryFrontendName(opts.Name))
	serviceName := serviceNameQueryFrontendHTTP(opts.Name)
	lokiEndpoint := lokiPodMetricsEndpoint(opts.Name, lokiHTTPPortName, serviceName, opts.Namespace, opts.Gates.ServiceMonitorTLSEndpoints)

	return newPodMonitor(opts.Namespace, serviceMonitorName, l, lokiEndpoint)
}

// NewIndexGatewayPodMonitor creates a k8s service monitor for the index-gateway component
func NewIndexGatewayPodMonitor(opts Options) *monitoringv1.PodMonitor {
	l := ComponentLabels(LabelIndexGatewayComponent, opts.Name)

	serviceMonitorName := serviceMonitorName(IndexGatewayName(opts.Name))
	serviceName := serviceNameIndexGatewayHTTP(opts.Name)
	lokiEndpoint := lokiPodMetricsEndpoint(opts.Name, lokiHTTPPortName, serviceName, opts.Namespace, opts.Gates.ServiceMonitorTLSEndpoints)

	return newPodMonitor(opts.Namespace, serviceMonitorName, l, lokiEndpoint)
}

// NewRulerPodMonitor creates a k8s service monitor for the ruler component
func NewRulerPodMonitor(opts Options) *monitoringv1.PodMonitor {
	l := ComponentLabels(LabelRulerComponent, opts.Name)

	serviceMonitorName := serviceMonitorName(RulerName(opts.Name))
	serviceName := serviceNameRulerHTTP(opts.Name)
	lokiEndpoint := lokiPodMetricsEndpoint(opts.Name, lokiHTTPPortName, serviceName, opts.Namespace, opts.Gates.ServiceMonitorTLSEndpoints)

	return newPodMonitor(opts.Namespace, serviceMonitorName, l, lokiEndpoint)
}

// NewGatewayPodMonitor creates a k8s service monitor for the lokistack-gateway component
func NewGatewayPodMonitor(opts Options) *monitoringv1.PodMonitor {
	l := ComponentLabels(LabelGatewayComponent, opts.Name)

	gatewayName := GatewayName(opts.Name)
	serviceMonitorName := serviceMonitorName(gatewayName)
	serviceName := serviceNameGatewayHTTP(opts.Name)
	gwEndpoint := gatewayPodMetricsEndpoint(gatewayName, gatewayInternalPortName, serviceName, opts.Namespace, opts.Gates.ServiceMonitorTLSEndpoints)

	sm := newPodMonitor(opts.Namespace, serviceMonitorName, l, gwEndpoint)

	if opts.Stack.Tenants != nil {
		if err := configureGatewayPodMonitorForMode(sm, opts); err != nil {
			return sm
		}
	}

	return sm
}

func newPodMonitor(namespace, monitorName string, labels labels.Set, endpoint monitoringv1.PodMetricsEndpoint) *monitoringv1.PodMonitor {
	return &monitoringv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      monitorName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: monitoringv1.PodMonitorSpec{
			PodTargetLabels: []string{
				kubernetesInstanceLabel,
				kubernetesComponentLabel,
			},
			PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
				endpoint,
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{namespace},
			},
		},
	}
}
