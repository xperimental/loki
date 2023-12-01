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
func SetStorageSchemaStatus(ctx context.Context, k k8s.Client, req ctrl.Request, schemas []lokiv1.ObjectStorageSchema, now time.Time, retentionDays int) error {
	var s lokiv1.LokiStack
	if err := k.Get(ctx, req.NamespacedName, &s); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return kverrors.Wrap(err, "failed to lookup lokistack", "name", req.NamespacedName)
	}

	modified := s.DeepCopy()
	modified.Status.Storage.Schemas = convertToSchemaStatus(schemas, now, retentionDays)

	return k.Status().Update(ctx, modified)
}

func convertToSchemaStatus(schemas []lokiv1.ObjectStorageSchema, now time.Time, retentionDays int) []lokiv1.ObjectStorageSchemaStatus {
	retentionDuration := dayDuration * time.Duration(retentionDays)

	statuses := make([]lokiv1.ObjectStorageSchemaStatus, 0, len(schemas))
	for i, s := range schemas {
		effectiveDate, _ := s.EffectiveDate.UTCTime()
		var endDate time.Time
		if i+1 < len(schemas) {
			next := schemas[i+1]
			nextEffective, _ := next.EffectiveDate.UTCTime()
			endDate = nextEffective.Add(-dayDuration)
		}

		statuses = append(statuses, lokiv1.ObjectStorageSchemaStatus{
			ObjectStorageSchema: s,
			EndDate:             optionalEffectiveDate(endDate),
			Status:              schemaStatus(retentionDuration, now, effectiveDate, endDate),
		})
	}
	return statuses
}

func optionalEffectiveDate(date time.Time) lokiv1.StorageSchemaEffectiveDate {
	if date.IsZero() {
		return ""
	}

	return lokiv1.StorageSchemaEffectiveDate(date.Format(lokiv1.StorageSchemaEffectiveDateFormat))
}

func schemaStatus(retentionDuration time.Duration, now, effectiveDate, endDate time.Time) lokiv1.ObjectStorageSchemaStatusType {
	if !endDate.IsZero() && retentionDuration > 0 && now.Sub(endDate) > retentionDuration {
		return lokiv1.SchemaStatusObsolete
	}

	if now.Before(effectiveDate) {
		return lokiv1.SchemaStatusFuture
	}

	return lokiv1.SchemaStatusInUse
}
