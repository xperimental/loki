package status

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/external/k8s"
)

const (
	warningObsoleteSchemaReason  = "WarningObsoleteSchema"
	warningObsoleteSchemaMessage = "The schema configuration contains one or more schemas that are not in use anymore due to retention settings."

	warningOldSchemaVersionReason  = "WarningOldSchemaVersion"
	warningOldSchemaVersionMessage = "The schema configuration contains one or more schemas that do not use the most recent version."

	warningFutureOldSchemaVersionReason  = "WarningFutureOldSchemaVersion"
	warningFutureOldSchemaVersionMessage = "The schema configuration contains future schemas, that do not use the most recent version."
)

func createWarning(reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    string(lokiv1.ConditionWarning),
		Reason:  reason,
		Message: message,
	}
}

func generateWarnings(ctx context.Context, cs *lokiv1.LokiStackComponentStatus, k k8s.Client, req ctrl.Request, stack *lokiv1.LokiStack, now time.Time) ([]metav1.Condition, error) {
	hasObsoleteSchema := false
	hasOldSchemaVersion := false
	hasFutureOldSchemaVersion := false
	for _, schema := range stack.Status.Storage.Schemas {
		if schema.Status == lokiv1.SchemaStatusObsolete {
			hasObsoleteSchema = true
		}

		if schema.Version != lokiv1.ObjectStorageSchemaV13 {
			hasOldSchemaVersion = true

			if schema.Status == lokiv1.SchemaStatusFuture {
				hasFutureOldSchemaVersion = true
			}
		}
	}

	warnings := make([]metav1.Condition, 0)
	if hasObsoleteSchema {
		warnings = append(warnings, createWarning(
			warningObsoleteSchemaReason,
			warningObsoleteSchemaMessage,
		))
	}

	if hasOldSchemaVersion {
		warnings = append(warnings, createWarning(
			warningOldSchemaVersionReason,
			warningOldSchemaVersionMessage,
		))
	}

	if hasFutureOldSchemaVersion {
		warnings = append(warnings, createWarning(
			warningFutureOldSchemaVersionReason,
			warningFutureOldSchemaVersionMessage,
		))
	}
	return warnings, nil
}
