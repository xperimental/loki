package manifests

import (
	"fmt"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/manifests/internal"
	"github.com/grafana/loki/operator/internal/manifests/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	"path"
)

func extractComponentSpec(opts *lokiv1.LokiTemplateSpec, componentName string) *lokiv1.LokiComponentSpec {
	var componentSpec *lokiv1.LokiComponentSpec
	if opts != nil {
		switch componentName {
		case LabelCompactorComponent:
			componentSpec = opts.Compactor
		case LabelDistributorComponent:
			componentSpec = opts.Distributor
		case LabelIngesterComponent:
			componentSpec = opts.Ingester
		case LabelQuerierComponent:
			componentSpec = opts.Querier
		case LabelQueryFrontendComponent:
			componentSpec = opts.QueryFrontend
		case LabelIndexGatewayComponent:
			componentSpec = opts.IndexGateway
		case LabelRulerComponent:
			componentSpec = opts.Ruler
		case LabelGatewayComponent:
			componentSpec = opts.Gateway
		default:
			panic(fmt.Sprintf("extractComponentSpec: unknown componentName: %s", componentName))
		}
	}

	if componentSpec == nil {
		return &lokiv1.LokiComponentSpec{}
	}

	return componentSpec
}

func deploymentResources(requirements internal.ComponentResources, componentName string) corev1.ResourceRequirements {
	switch componentName {
	case LabelQuerierComponent:
		return requirements.Querier
	case LabelDistributorComponent:
		return requirements.Distributor
	case LabelQueryFrontendComponent:
		return requirements.QueryFrontend
	default:
		panic(fmt.Sprintf("deploymentResources: unknown deployment for resource requirements: %s", componentName))
	}
}

func newDeployment(opts Options, componentName string) *appsv1.Deployment {
	componentSpec := extractComponentSpec(opts.Stack.Template, componentName)
	resourceRequirements := deploymentResources(opts.ResourceRequirements, componentName)

	l := ComponentLabels(componentName, opts.Name)
	a := commonAnnotations(opts.ConfigSHA1, opts.ObjectStorage.SecretSHA1, opts.CertRotationRequiredAt)
	podSpec := corev1.PodSpec{
		Affinity: configureAffinity(componentName, opts.Name, opts.Gates.DefaultNodeAffinity, componentSpec),
		Volumes: []corev1.Volume{
			{
				Name: configVolumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						DefaultMode: &defaultConfigMapMode,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: lokiConfigMapName(opts.Name),
						},
					},
				},
			},
		},
		Containers: []corev1.Container{
			{
				Image: opts.Image,
				Name:  fmt.Sprintf("loki-%s", componentName),
				Resources: corev1.ResourceRequirements{
					Limits:   resourceRequirements.Limits,
					Requests: resourceRequirements.Requests,
				},
				Args: []string{
					fmt.Sprintf("-target=%s", componentName),
					fmt.Sprintf("-config.file=%s", path.Join(config.LokiConfigMountDir, config.LokiConfigFileName)),
					fmt.Sprintf("-runtime-config.file=%s", path.Join(config.LokiConfigMountDir, config.LokiRuntimeConfigFileName)),
					"-config.expand-env=true",
				},
				ReadinessProbe: lokiReadinessProbe(),
				LivenessProbe:  lokiLivenessProbe(),
				Ports: []corev1.ContainerPort{
					{
						Name:          lokiHTTPPortName,
						ContainerPort: httpPort,
						Protocol:      protocolTCP,
					},
					{
						Name:          lokiGRPCPortName,
						ContainerPort: grpcPort,
						Protocol:      protocolTCP,
					},
					{
						Name:          lokiGossipPortName,
						ContainerPort: gossipPort,
						Protocol:      protocolTCP,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      configVolumeName,
						ReadOnly:  false,
						MountPath: config.LokiConfigMountDir,
					},
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: "File",
				ImagePullPolicy:          "IfNotPresent",
			},
		},
	}

	if opts.Stack.Template != nil && componentSpec != nil {
		podSpec.Tolerations = componentSpec.Tolerations
		podSpec.NodeSelector = componentSpec.NodeSelector
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", opts.Name, componentName),
			Labels: l,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(componentSpec.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.Merge(l, GossipLabels()),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        fmt.Sprintf("loki-%s-%s", componentName, opts.Name),
					Labels:      labels.Merge(l, GossipLabels()),
					Annotations: a,
				},
				Spec: podSpec,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
		},
	}
}
