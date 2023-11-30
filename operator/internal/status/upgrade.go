package status

import (
	"context"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
)

const (
	dayDuration = 24 * time.Hour
)

func generateSchemaUpgrade(ctx context.Context, stack *lokiv1.LokiStack, now time.Time) (*lokiv1.ProposedSchemaUpdate, error) {
	futureSchemaTime := now.Add(5 * dayDuration)
	applyTime := now.Add(3 * dayDuration)
	existingUpgrade := stack.Status.Storage.AutomaticUpgrade
	if existingUpgrade != nil && now.Before(existingUpgrade.UpgradeTime.Time) {
		// if existing upgrade is still in the future, return the same
		return existingUpgrade, nil
	}

	specSchemas := stack.Spec.Storage.Schemas
	sort.Slice(specSchemas, func(i, j int) bool {
		iDate, _ := specSchemas[i].EffectiveDate.UTCTime()
		jDate, _ := specSchemas[j].EffectiveDate.UTCTime()

		return iDate.Before(jDate)
	})

	futureIdx := -1
	for i, s := range specSchemas {
		date, _ := s.EffectiveDate.UTCTime()
		if now.Before(date) {
			futureIdx = i
			break
		}
	}

	if futureIdx == -1 {
		// the current configuration has no future configurations
		specSchemas = append(specSchemas, lokiv1.ObjectStorageSchema{
			Version:       lokiv1.ObjectStorageSchemaV13,
			EffectiveDate: lokiv1.StorageSchemaEffectiveDate(futureSchemaTime.UTC().Format(lokiv1.StorageSchemaEffectiveDateFormat)),
		})

		return &lokiv1.ProposedSchemaUpdate{
			UpgradeTime: metav1.NewTime(applyTime),
			Schemas:     specSchemas,
		}, nil
	}

	if specSchemas[futureIdx].Version == lokiv1.ObjectStorageSchemaV13 {
		// First future spec is already current version
		if futureIdx+1 == len(specSchemas) {
			// no further specs are present -> everything is fine
			return nil, nil
		}
	}

	// there are future specs, but they do not update to the latest version -> cut them off and apply a new one
	specSchemas = specSchemas[:futureIdx]
	specSchemas = append(specSchemas, lokiv1.ObjectStorageSchema{
		Version:       lokiv1.ObjectStorageSchemaV13,
		EffectiveDate: lokiv1.StorageSchemaEffectiveDate(futureSchemaTime.UTC().Format(lokiv1.StorageSchemaEffectiveDateFormat)),
	})

	return &lokiv1.ProposedSchemaUpdate{
		UpgradeTime: metav1.NewTime(applyTime),
		Schemas:     specSchemas,
	}, nil
}
