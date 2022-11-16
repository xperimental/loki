package lokistack

import (
	"context"
	"time"

	"github.com/ViaQ/logerr/v2/kverrors"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/external/k8s"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterResourceDiscoveredAtKey = "loki.grafana.com/clusterResourceDiscoveredAt"
	rulesDiscoveredAtKey           = "loki.grafana.com/rulesDiscoveredAt"
)

// AnnotateForRequiredClusterResource adds/updates the `loki.grafana.com/clusterResourceDiscoveredAt` annotation
// to all instance of LokiStack on all namespaces to trigger the reconciliation loop.
func AnnotateForRequiredClusterResource(ctx context.Context, k k8s.Client) error {
	return annotateAll(ctx, k, clusterResourceDiscoveredAtKey)
}

// AnnotateForDiscoveredRules adds/updates the `loki.grafana.com/rulesDiscoveredAt` annotation
// to all instance of LokiStack on all namespaces to trigger the reconciliation loop.
func AnnotateForDiscoveredRules(ctx context.Context, k k8s.Client) error {
	return annotateAll(ctx, k, rulesDiscoveredAtKey)
}

func annotateAll(ctx context.Context, k k8s.Client, key string) error {
	var stacks lokiv1.LokiStackList
	err := k.List(ctx, &stacks, client.MatchingLabelsSelector{Selector: labels.Everything()})
	if err != nil {
		return kverrors.Wrap(err, "failed to list any lokistack instances", "req")
	}

	for _, s := range stacks.Items {
		ss := s.DeepCopy()
		if ss.Annotations == nil {
			ss.Annotations = make(map[string]string)
		}

		ss.Annotations[key] = time.Now().UTC().Format(time.RFC3339)

		if err := k.Update(ctx, ss); err != nil {
			return kverrors.Wrap(err, "failed to update lokistack annotation", "name", ss.Name, "namespace", ss.Namespace, "key", key)
		}
	}

	return nil
}
