package status

import (
	"context"
	"time"

	"github.com/ViaQ/logerr/v2/kverrors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/external/k8s"
)

// SetStorageSchemaStatus updates the storage status component
func SetStorageSchemaStatus(ctx context.Context, k k8s.Client, req ctrl.Request, schemas []lokiv1.ObjectStorageSchema, now time.Time) error {
	var s lokiv1.LokiStack
	if err := k.Get(ctx, req.NamespacedName, &s); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return kverrors.Wrap(err, "failed to lookup lokistack", "name", req.NamespacedName)
	}

	modified := s.DeepCopy()
	modified.Status.Storage.Schemas = convertToSchemaStatus(schemas, now)

	return k.Status().Update(ctx, modified)
}

func convertToSchemaStatus(schemas []lokiv1.ObjectStorageSchema, now time.Time) []lokiv1.ObjectStorageSchemaStatus {
	statuses := make([]lokiv1.ObjectStorageSchemaStatus, 0, len(schemas))
	for _, s := range schemas {
		statuses = append(statuses, lokiv1.ObjectStorageSchemaStatus{
			ObjectStorageSchema: s,
			Status:              schemaStatus(s, now),
		})
	}
	return statuses
}

func schemaStatus(s lokiv1.ObjectStorageSchema, now time.Time) lokiv1.ObjectStorageSchemaStatusType {
	effectiveDate, _ := s.EffectiveDate.UTCTime()
	if now.Before(effectiveDate) {
		return lokiv1.SchemaStatusFuture
	}

	return lokiv1.SchemaStatusInUse
}
