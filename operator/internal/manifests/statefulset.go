package manifests

import (
	"fmt"
	"github.com/grafana/loki/operator/internal/manifests/internal"
	"github.com/grafana/loki/operator/internal/manifests/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	"path"
)

func statefulSetResources(requirements internal.ComponentResources, componentName string) internal.ResourceRequirements {
	switch componentName {
	case LabelIndexGatewayComponent:
		return requirements.IndexGateway
	case LabelIngesterComponent:
		return requirements.Ingester
	case LabelCompactorComponent:
		return requirements.Compactor
	case LabelRulerComponent:
		return requirements.Ruler
	default:
		panic(fmt.Sprintf("statefulSetResources: unknown component name: %s", componentName))
	}
}

func newStatefulSet(opts Options, componentName string, hasGossipPort, hasWalVolume bool) *appsv1.StatefulSet {
	componentSpec := extractComponentSpec(opts.Stack.Template, componentName)
	resourceRequirements := statefulSetResources(opts.ResourceRequirements, componentName)

	l := ComponentLabels(componentName, opts.Name)
	a := commonAnnotations(opts.ConfigSHA1, opts.ObjectStorage.SecretSHA1, opts.CertRotationRequiredAt)
	podSpec := corev1.PodSpec{
		Tolerations:  componentSpec.Tolerations,
		NodeSelector: componentSpec.NodeSelector,
		Affinity:     configureAffinity(componentName, opts.Name, opts.Gates.DefaultNodeAffinity, componentSpec),
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
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      configVolumeName,
						ReadOnly:  false,
						MountPath: config.LokiConfigMountDir,
					},
					{
						Name:      storageVolumeName,
						ReadOnly:  false,
						MountPath: dataDirectory,
					},
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: "File",
				ImagePullPolicy:          "IfNotPresent",
			},
		},
	}

	if hasGossipPort {
		podSpec.Containers[0].Ports = append(podSpec.Containers[0].Ports, corev1.ContainerPort{
			Name:          lokiGossipPortName,
			ContainerPort: gossipPort,
			Protocol:      protocolTCP,
		})
	}

	stateFulSet := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", opts.Name, componentName),
			Labels: l,
		},
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy:  appsv1.OrderedReadyPodManagement,
			RevisionHistoryLimit: pointer.Int32(10),
			Replicas:             pointer.Int32(componentSpec.Replicas),
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
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: l,
						Name:   storageVolumeName,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							// TODO: should we verify that this is possible with the given storage class first?
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resourceRequirements.PVCSize,
							},
						},
						StorageClassName: pointer.String(opts.Stack.StorageClassName),
						VolumeMode:       &volumeFileSystemMode,
					},
				},
			},
		},
	}

	if hasWalVolume {
		stateFulSet.Spec.Template.Spec.Containers[0].VolumeMounts = append(stateFulSet.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      walVolumeName,
			ReadOnly:  false,
			MountPath: walDirectory,
		})

		stateFulSet.Spec.VolumeClaimTemplates = append(stateFulSet.Spec.VolumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Labels: l,
				Name:   walVolumeName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					// TODO: should we verify that this is possible with the given storage class first?
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: opts.ResourceRequirements.WALStorage.PVCSize,
					},
				},
				StorageClassName: pointer.String(opts.Stack.StorageClassName),
				VolumeMode:       &volumeFileSystemMode,
			},
		})
	}

	return stateFulSet
}
