package manifests

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/loki/operator/internal/manifests/storage"
)

// BuildIndexGateway returns a list of k8s objects for Loki IndexGateway
func BuildIndexGateway(opts Options) ([]client.Object, error) {
	statefulSet := NewIndexGatewayStatefulSet(opts)
	if opts.Gates.HTTPEncryption {
		if err := configureIndexGatewayHTTPServicePKI(statefulSet, opts); err != nil {
			return nil, err
		}
	}

	if err := storage.ConfigureStatefulSet(statefulSet, opts.ObjectStorage); err != nil {
		return nil, err
	}

	if opts.Gates.GRPCEncryption {
		if err := configureIndexGatewayGRPCServicePKI(statefulSet, opts); err != nil {
			return nil, err
		}
	}

	if opts.Gates.HTTPEncryption || opts.Gates.GRPCEncryption {
		caBundleName := signingCABundleName(opts.Name)
		if err := configureServiceCA(&statefulSet.Spec.Template.Spec, caBundleName); err != nil {
			return nil, err
		}
	}

	if opts.Gates.RestrictedPodSecurityStandard {
		if err := configurePodSpecForRestrictedStandard(&statefulSet.Spec.Template.Spec); err != nil {
			return nil, err
		}
	}

	if err := configureHashRingEnv(&statefulSet.Spec.Template.Spec, opts); err != nil {
		return nil, err
	}

	if err := configureProxyEnv(&statefulSet.Spec.Template.Spec, opts); err != nil {
		return nil, err
	}

	if err := configureReplication(&statefulSet.Spec.Template, opts.Stack.Replication, LabelIndexGatewayComponent, opts.Name); err != nil {
		return nil, err
	}

	return []client.Object{
		statefulSet,
		NewIndexGatewayGRPCService(opts),
		NewIndexGatewayHTTPService(opts),
		NewIndexGatewayPodDisruptionBudget(opts),
	}, nil
}

// NewIndexGatewayStatefulSet creates a statefulset object for an index-gateway
func NewIndexGatewayStatefulSet(opts Options) *appsv1.StatefulSet {
	return newStatefulSet(opts, LabelIndexGatewayComponent, false, false)
}

// NewIndexGatewayGRPCService creates a k8s service for the index-gateway GRPC endpoint
func NewIndexGatewayGRPCService(opts Options) *corev1.Service {
	serviceName := serviceNameIndexGatewayGRPC(opts.Name)
	labels := ComponentLabels(LabelIndexGatewayComponent, opts.Name)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName,
			Labels: labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       lokiGRPCPortName,
					Port:       grpcPort,
					Protocol:   protocolTCP,
					TargetPort: intstr.IntOrString{IntVal: grpcPort},
				},
			},
			Selector: labels,
		},
	}
}

// NewIndexGatewayHTTPService creates a k8s service for the index-gateway HTTP endpoint
func NewIndexGatewayHTTPService(opts Options) *corev1.Service {
	serviceName := serviceNameIndexGatewayHTTP(opts.Name)
	labels := ComponentLabels(LabelIndexGatewayComponent, opts.Name)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName,
			Labels: labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       lokiHTTPPortName,
					Port:       httpPort,
					Protocol:   protocolTCP,
					TargetPort: intstr.IntOrString{IntVal: httpPort},
				},
			},
			Selector: labels,
		},
	}
}

// NewIndexGatewayPodDisruptionBudget returns a PodDisruptionBudget for the LokiStack
// index-gateway pods.
func NewIndexGatewayPodDisruptionBudget(opts Options) *policyv1.PodDisruptionBudget {
	l := ComponentLabels(LabelIndexGatewayComponent, opts.Name)
	ma := intstr.FromInt(1)
	return &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: policyv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    l,
			Name:      IndexGatewayName(opts.Name),
			Namespace: opts.Namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: l,
			},
			MinAvailable: &ma,
		},
	}
}

func configureIndexGatewayHTTPServicePKI(statefulSet *appsv1.StatefulSet, opts Options) error {
	serviceName := serviceNameIndexGatewayHTTP(opts.Name)
	return configureHTTPServicePKI(&statefulSet.Spec.Template.Spec, serviceName)
}

func configureIndexGatewayGRPCServicePKI(sts *appsv1.StatefulSet, opts Options) error {
	serviceName := serviceNameIndexGatewayGRPC(opts.Name)
	return configureGRPCServicePKI(&sts.Spec.Template.Spec, serviceName)
}
