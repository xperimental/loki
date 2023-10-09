package manifests

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildQueryFrontend returns a list of k8s objects for Loki QueryFrontend
func BuildQueryFrontend(opts Options) ([]client.Object, error) {
	deployment := NewQueryFrontendDeployment(opts)
	if opts.Gates.HTTPEncryption {
		if err := configureQueryFrontendHTTPServicePKI(deployment, opts); err != nil {
			return nil, err
		}
	}

	if opts.Gates.GRPCEncryption {
		if err := configureQueryFrontendGRPCServicePKI(deployment, opts); err != nil {
			return nil, err
		}
	}

	if opts.Gates.HTTPEncryption || opts.Gates.GRPCEncryption {
		caBundleName := signingCABundleName(opts.Name)
		if err := configureServiceCA(&deployment.Spec.Template.Spec, caBundleName); err != nil {
			return nil, err
		}
	}

	if opts.Gates.RestrictedPodSecurityStandard {
		if err := configurePodSpecForRestrictedStandard(&deployment.Spec.Template.Spec); err != nil {
			return nil, err
		}
	}

	if err := configureHashRingEnv(&deployment.Spec.Template.Spec, opts); err != nil {
		return nil, err
	}

	if err := configureProxyEnv(&deployment.Spec.Template.Spec, opts); err != nil {
		return nil, err
	}

	if err := configureReplication(&deployment.Spec.Template, opts.Stack.Replication, LabelQueryFrontendComponent, opts.Name); err != nil {
		return nil, err
	}

	return []client.Object{
		deployment,
		NewQueryFrontendGRPCService(opts),
		NewQueryFrontendHTTPService(opts),
		NewQueryFrontendPodDisruptionBudget(opts),
	}, nil
}

// NewQueryFrontendDeployment creates a deployment object for a query-frontend
func NewQueryFrontendDeployment(opts Options) *appsv1.Deployment {
	deployment := newDeployment(opts, LabelQueryFrontendComponent, false)

	// The frontend will only return ready once a querier has connected to it.
	// Because the service used for connecting the querier to the frontend only lists ready
	// instances there's sequencing issue. For now, we re-use the liveness-probe path
	// for the readiness-probe as a workaround.
	deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path = lokiLivenessPath

	return deployment
}

// NewQueryFrontendGRPCService creates a k8s service for the query-frontend GRPC endpoint
func NewQueryFrontendGRPCService(opts Options) *corev1.Service {
	serviceName := serviceNameQueryFrontendGRPC(opts.Name)
	labels := ComponentLabels(LabelQueryFrontendComponent, opts.Name)

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

// NewQueryFrontendHTTPService creates a k8s service for the query-frontend HTTP endpoint
func NewQueryFrontendHTTPService(opts Options) *corev1.Service {
	serviceName := serviceNameQueryFrontendHTTP(opts.Name)
	labels := ComponentLabels(LabelQueryFrontendComponent, opts.Name)

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

// NewQueryFrontendPodDisruptionBudget returns a PodDisruptionBudget for the LokiStack
// query-frontend pods.
func NewQueryFrontendPodDisruptionBudget(opts Options) *policyv1.PodDisruptionBudget {
	l := ComponentLabels(LabelQueryFrontendComponent, opts.Name)
	ma := intstr.FromInt(1)
	return &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: policyv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    l,
			Name:      QueryFrontendName(opts.Name),
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

func configureQueryFrontendHTTPServicePKI(deployment *appsv1.Deployment, opts Options) error {
	serviceName := serviceNameQueryFrontendHTTP(opts.Name)
	return configureHTTPServicePKI(&deployment.Spec.Template.Spec, serviceName)
}

func configureQueryFrontendGRPCServicePKI(deployment *appsv1.Deployment, opts Options) error {
	serviceName := serviceNameQueryFrontendGRPC(opts.Name)
	return configureGRPCServicePKI(&deployment.Spec.Template.Spec, serviceName)
}
