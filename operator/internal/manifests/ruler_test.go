package manifests

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/manifests/openshift"
	"github.com/grafana/loki/operator/internal/manifests/storage"
)

func TestNewRulerStatefulSet_HasTemplateConfigHashAnnotation(t *testing.T) {
	ss := NewRulerStatefulSet(Options{
		Name:       "abcd",
		Namespace:  "efgh",
		ConfigSHA1: "deadbeef",
		Stack: lokiv1.LokiStackSpec{
			StorageClassName: "standard",
			Template: &lokiv1.LokiTemplateSpec{
				Ruler: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
			},
		},
	})

	annotations := ss.Spec.Template.Annotations
	require.Contains(t, annotations, AnnotationLokiConfigHash)
	require.Equal(t, annotations[AnnotationLokiConfigHash], "deadbeef")
}

func TestNewRulerStatefulSet_HasTemplateObjectStoreHashAnnotation(t *testing.T) {
	ss := NewRulerStatefulSet(Options{
		Name:      "abcd",
		Namespace: "efgh",
		ObjectStorage: storage.Options{
			SecretSHA1: "deadbeef",
		},
		Stack: lokiv1.LokiStackSpec{
			StorageClassName: "standard",
			Template: &lokiv1.LokiTemplateSpec{
				Ruler: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
			},
		},
	})

	annotations := ss.Spec.Template.Annotations
	require.Contains(t, annotations, AnnotationLokiObjectStoreHash)
	require.Equal(t, annotations[AnnotationLokiObjectStoreHash], "deadbeef")
}

func TestNewRulerStatefulSet_HasTemplateCertRotationRequiredAtAnnotation(t *testing.T) {
	ss := NewRulerStatefulSet(Options{
		Name:                   "abcd",
		Namespace:              "efgh",
		CertRotationRequiredAt: "deadbeef",
		Stack: lokiv1.LokiStackSpec{
			StorageClassName: "standard",
			Template: &lokiv1.LokiTemplateSpec{
				Ruler: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
			},
		},
	})

	annotations := ss.Spec.Template.Annotations
	require.Contains(t, annotations, AnnotationCertRotationRequiredAt)
	require.Equal(t, annotations[AnnotationCertRotationRequiredAt], "deadbeef")
}

func TestBuildRuler_HasExtraObjectsForTenantMode(t *testing.T) {
	objs, err := BuildRuler(Options{
		Name:      "abcd",
		Namespace: "efgh",
		OpenShiftOptions: openshift.Options{
			BuildOpts: openshift.BuildOptions{
				LokiStackName:      "abc",
				LokiStackNamespace: "efgh",
			},
		},
		Stack: lokiv1.LokiStackSpec{
			Template: &lokiv1.LokiTemplateSpec{
				Ruler: &lokiv1.LokiComponentSpec{
					Replicas: rand.Int31(),
				},
			},
			Rules: &lokiv1.RulesSpec{
				Enabled: true,
			},
			Tenants: &lokiv1.TenantsSpec{
				Mode: lokiv1.OpenshiftLogging,
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, objs, 7)
}

func TestNewRulerStatefulSet_SelectorMatchesLabels(t *testing.T) {
	// You must set the .spec.selector field of a StatefulSet to match the labels of
	// its .spec.template.metadata.labels. Prior to Kubernetes 1.8, the
	// .spec.selector field was defaulted when omitted. In 1.8 and later versions,
	// failing to specify a matching Pod Selector will result in a validation error
	// during StatefulSet creation.
	// See https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#pod-selector
	sts := NewRulerStatefulSet(Options{
		Name:      "abcd",
		Namespace: "efgh",
		Stack: lokiv1.LokiStackSpec{
			StorageClassName: "standard",
			Template: &lokiv1.LokiTemplateSpec{
				Ruler: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
			},
		},
	})

	l := sts.Spec.Template.GetObjectMeta().GetLabels()
	for key, value := range sts.Spec.Selector.MatchLabels {
		require.Contains(t, l, key)
		require.Equal(t, l[key], value)
	}
}

func TestNewRulerStatefulSet_MountsRulesInPerTenantIDSubDirectories(t *testing.T) {
	sts := NewRulerStatefulSet(Options{
		Name:      "abcd",
		Namespace: "efgh",
		Stack: lokiv1.LokiStackSpec{
			StorageClassName: "standard",
			Template: &lokiv1.LokiTemplateSpec{
				Ruler: &lokiv1.LokiComponentSpec{
					Replicas: 1,
				},
			},
		},
		Tenants: Tenants{
			Configs: map[string]TenantConfig{
				"tenant-a": {RuleFiles: []string{"rule-a-alerts.yaml", "rule-b-recs.yaml"}},
				"tenant-b": {RuleFiles: []string{"rule-a-alerts.yaml", "rule-b-recs.yaml"}},
			},
		},
	})

	vs := sts.Spec.Template.Spec.Volumes

	var (
		volumeNames []string
		volumeItems []corev1.KeyToPath
	)
	for _, v := range vs {
		volumeNames = append(volumeNames, v.Name)
		volumeItems = append(volumeItems, v.ConfigMap.Items...)
	}

	require.NotEmpty(t, volumeNames)
	require.NotEmpty(t, volumeItems)
}
