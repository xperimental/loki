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

var StorageSchemaOutOfRetention = "Old object storage schema v11/v12 is out of retention"

// SetStorageSchemaStatus updates the storage status component
func SetStorageSchemaStatus(ctx context.Context, k k8s.Client, req ctrl.Request, schemas []lokiv1.ObjectStorageSchema) error {
	var s lokiv1.LokiStack
	utcTime := time.Now()
	cutoff := utcTime.Add(lokiv1.StorageSchemaUpdateBuffer)

	var oldSchemas []lokiv1.ObjectStorageSchema

	if err := k.Get(ctx, req.NamespacedName, &s); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return kverrors.Wrap(err, "failed to lookup lokistack", "name", req.NamespacedName)
	}

	s.Status.Storage = lokiv1.LokiStackStorageStatus{
		Schemas: schemas,
	}

	for _, sc := range schemas {
		date, err := sc.EffectiveDate.UTCTime()
		if err != nil {
			return kverrors.Wrap(err, "failed to parse effective date")
		}
		if sc.Version == lokiv1.ObjectStorageSchemaV13 && date.Before(cutoff) {
			continue
		} else {
			oldSchemas = append(oldSchemas, sc)
		}
	}

	if len(oldSchemas) > 0 {
		s.Status.Storage.SchemaStatus = StorageSchemaOutOfRetention
	}

	return k.Status().Update(ctx, &s)
}
