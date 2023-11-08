package status

import (
	"context"
	"errors"
	"time"

	"github.com/ViaQ/logerr/v2/kverrors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/external/k8s"
)

var (
	StorageSchemaOutOfRetention = "Old object storage schema V11/V12 is out of retention"
	StorageSchemaNeedsUpgrade   = "Consider upgrading to schema V13 to use TSDB shipper"
	WarningError                = errors.New(string(lokiv1.ConditionWarning))
)

// SetStorageSchemaStatus updates the storage status component
func SetStorageSchemaStatus(ctx context.Context, k k8s.Client, req ctrl.Request, schemas []lokiv1.ObjectStorageSchema) error {
	var s lokiv1.LokiStack
	utcTime := time.Now()
	cutoff := utcTime.Add(lokiv1.StorageSchemaUpdateBuffer)

	// schemaVersionMap maps the existing schema versions to a boolean value
	// that flags whether there is an applied v13 schema or not
	schemaVersionMap := make(map[lokiv1.ObjectStorageSchemaVersion]bool)

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
			schemaVersionMap[sc.Version] = true
			continue
		} else {
			schemaVersionMap[sc.Version] = false
			oldSchemas = append(oldSchemas, sc)
		}
	}

	// TODO: refactor schemaVersionMap to a slice once we upgrade to go 1.21 and use the slices package
	if len(oldSchemas) > 0 {
		if schemaVersionMap[lokiv1.ObjectStorageSchemaV13] {
			s.Status.Storage.SchemaStatus = StorageSchemaOutOfRetention
		} else {
			s.Status.Storage.SchemaStatus = StorageSchemaNeedsUpgrade
			if err := k.Status().Update(ctx, &s); err != nil {
				return err
			}

			return WarningError
		}
	}

	return k.Status().Update(ctx, &s)
}
