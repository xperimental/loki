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

// BuildIngester builds the k8s objects required to run Loki Ingester
func BuildIngester(opts Options) ([]client.Object, error) {
	statefulSet := NewIngesterStatefulSet(opts)
	if opts.Gates.HTTPEncryption {
		if err := configureIngesterHTTPServicePKI(statefulSet, opts); err != nil {
			return nil, err
		}
	}

	if err := storage.ConfigureStatefulSet(statefulSet, opts.ObjectStorage); err != nil {
		return nil, err
	}

	if opts.Gates.GRPCEncryption {
		if err := configureIngesterGRPCServicePKI(statefulSet, opts); err != nil {
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

	if err := configureReplication(&statefulSet.Spec.Template, opts.Stack.Replication, LabelIngesterComponent, opts.Name); err != nil {
		return nil, err
	}

	return []client.Object{
		statefulSet,
		NewIngesterGRPCService(opts),
		NewIngesterHTTPService(opts),
		newIngesterPodDisruptionBudget(opts),
	}, nil
}

// NewIngesterStatefulSet creates a deployment object for an ingester
func NewIngesterStatefulSet(opts Options) *appsv1.StatefulSet {
	return newStatefulSet(opts, LabelIngesterComponent)
}

// NewIngesterGRPCService creates a k8s service for the ingester GRPC endpoint
func NewIngesterGRPCService(opts Options) *corev1.Service {
	labels := ComponentLabels(LabelIngesterComponent, opts.Name)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceNameIngesterGRPC(opts.Name),
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

// NewIngesterHTTPService creates a k8s service for the ingester HTTP endpoint
func NewIngesterHTTPService(opts Options) *corev1.Service {
	serviceName := serviceNameIngesterHTTP(opts.Name)
	labels := ComponentLabels(LabelIngesterComponent, opts.Name)

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

func configureIngesterHTTPServicePKI(statefulSet *appsv1.StatefulSet, opts Options) error {
	serviceName := serviceNameIngesterHTTP(opts.Name)
	return configureHTTPServicePKI(&statefulSet.Spec.Template.Spec, serviceName)
}

func configureIngesterGRPCServicePKI(sts *appsv1.StatefulSet, opts Options) error {
	serviceName := serviceNameIngesterGRPC(opts.Name)
	return configureGRPCServicePKI(&sts.Spec.Template.Spec, serviceName)
}

// newIngesterPodDisruptionBudget returns a PodDisruptionBudget for the LokiStack
// Ingester pods.
func newIngesterPodDisruptionBudget(opts Options) *policyv1.PodDisruptionBudget {
	l := ComponentLabels(LabelIngesterComponent, opts.Name)
	// Default to 1 if not defined in ResourceRequirementsTable for a given size
	mu := intstr.FromInt(1)
	if opts.ResourceRequirements.Ingester.PDBMinAvailable > 0 {
		mu = intstr.FromInt(opts.ResourceRequirements.Ingester.PDBMinAvailable)
	}
	return &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: policyv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    l,
			Name:      IngesterName(opts.Name),
			Namespace: opts.Namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: l,
			},
			MinAvailable: &mu,
		},
	}
}
