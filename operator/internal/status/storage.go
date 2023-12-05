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

var (
	StorageSchemaOutOfRetention = "Old object storage schema is out of retention"
	StorageSchemaNeedsUpgrade   = "Consider upgrading to schema V13 to use TSDB shipper"
)

// SetStorageSchemaStatus updates the storage status component
func SetStorageSchemaStatus(ctx context.Context, k k8s.Client, req ctrl.Request, schemas []lokiv1.ObjectStorageSchema) error {
	var s lokiv1.LokiStack
	utcTime := time.Now()
	cutoff := utcTime.Add(lokiv1.StorageSchemaUpdateBuffer)

	// schemaVersionMap maps the existing schema versions to a boolean value
	// that flags whether there is an applied v13 schema or not
	schemaVersionMap := make(map[lokiv1.ObjectStorageSchemaVersion]bool)

	var oldSchemas []lokiv1.ObjectStorageStatusSchema

	if err := k.Get(ctx, req.NamespacedName, &s); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return kverrors.Wrap(err, "failed to lookup lokistack", "name", req.NamespacedName)
	}
	statusSchemas := storageSchemaToStatusSchema(schemas)
	s.Status.Storage = lokiv1.LokiStackStorageStatus{
		Schemas: statusSchemas,
	}

	for _, sc := range statusSchemas {
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

	if len(oldSchemas) > 0 {
		if schemaVersionMap[lokiv1.ObjectStorageSchemaV13] {
			if err := updateSchemaStatus(s.Status.Storage.Schemas, StorageSchemaOutOfRetention); err != nil {
				return kverrors.Wrap(err, "error updating schema status")
			}
		} else {
			if err := updateSchemaStatus(s.Status.Storage.Schemas, StorageSchemaNeedsUpgrade); err != nil {
				return kverrors.Wrap(err, "error updating schema status")
			}
		}
	}

	return k.Status().Update(ctx, &s)
}

func storageSchemaToStatusSchema(schemas []lokiv1.ObjectStorageSchema) []lokiv1.ObjectStorageStatusSchema {
	var statusSchemas []lokiv1.ObjectStorageStatusSchema

	for _, sc := range schemas {
		statusSchemas = append(statusSchemas, lokiv1.ObjectStorageStatusSchema{
			Version:       sc.Version,
			EffectiveDate: sc.EffectiveDate,
		})
	}

	return statusSchemas
}

func updateSchemaStatus(schemas []lokiv1.ObjectStorageStatusSchema, message string) error {
	for i, sc := range schemas {
		if sc.Version != lokiv1.ObjectStorageSchemaV13 {
			schemas[i].SchemaStatus = message
		}
	}
	return nil
}
