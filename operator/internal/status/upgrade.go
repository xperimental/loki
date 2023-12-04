package status

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
)

const (
	dayDuration          = 24 * time.Hour
	futureSchemaDuration = 5 * dayDuration
	applySchemaDuration  = 3 * dayDuration

	upgradeSchemaVersion = lokiv1.ObjectStorageSchemaV13
)

func generateSchemaUpgrade(ctx context.Context, stack *lokiv1.LokiStack, now time.Time) (*lokiv1.ProposedSchemaUpdate, error) {
	futureSchemaTime := now.Add(futureSchemaDuration)
	applyTime := now.Add(applySchemaDuration)
	existingUpgrade := stack.Status.Storage.AutomaticUpgrade
	if existingUpgrade != nil && now.Before(existingUpgrade.UpgradeTime.Time) {
		// if existing upgrade is still in the future, return the same
		return existingUpgrade, nil
	}

	statusSchemas := stack.Status.Storage.Schemas
	proposedSchemas := []lokiv1.ObjectStorageSchema{}

	futureIdx := -1
	skipped := 0
	for i, s := range statusSchemas {
		if s.Status == lokiv1.SchemaStatusObsolete {
			// remove obsolete schema configurations
			skipped++
			continue
		}

		proposedSchemas = append(proposedSchemas, s.ObjectStorageSchema)

		if s.Status == lokiv1.SchemaStatusFuture && futureIdx == -1 {
			futureIdx = i
		}
	}

	if futureIdx == -1 {
		// the current configuration has no future configurations
		if proposedSchemas[len(proposedSchemas)-1].Version == upgradeSchemaVersion {
			// last schema has the latest version, check for removed schemas
			if len(proposedSchemas) != len(statusSchemas) {
				return &lokiv1.ProposedSchemaUpdate{
					UpgradeTime: metav1.NewTime(applyTime),
					Schemas:     proposedSchemas,
				}, nil
			}

			// no schemas removed and current one is latest version, all fine
			return nil, nil
		}

		proposedSchemas = append(proposedSchemas, lokiv1.ObjectStorageSchema{
			Version:       upgradeSchemaVersion,
			EffectiveDate: lokiv1.StorageSchemaEffectiveDate(futureSchemaTime.UTC().Format(lokiv1.StorageSchemaEffectiveDateFormat)),
		})

		return &lokiv1.ProposedSchemaUpdate{
			UpgradeTime: metav1.NewTime(applyTime),
			Schemas:     proposedSchemas,
		}, nil
	}

	if statusSchemas[futureIdx].Version == upgradeSchemaVersion {
		// First future spec is already current version
		if futureIdx+1 == len(statusSchemas) {
			// no further specs are present -> everything is fine
			return nil, nil
		}
	}

	// there are future specs, but they do not update to the latest version -> cut them off and apply a new one
	proposedSchemas = proposedSchemas[:futureIdx-skipped]
	proposedSchemas = append(proposedSchemas, lokiv1.ObjectStorageSchema{
		Version:       upgradeSchemaVersion,
		EffectiveDate: lokiv1.StorageSchemaEffectiveDate(futureSchemaTime.UTC().Format(lokiv1.StorageSchemaEffectiveDateFormat)),
	})

	return &lokiv1.ProposedSchemaUpdate{
		UpgradeTime: metav1.NewTime(applyTime),
		Schemas:     proposedSchemas,
	}, nil
}
