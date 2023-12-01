package handlers

import (
	"context"
	"time"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/external/k8s"
)

func UpgradeStorageSchema(ctx context.Context, log logr.Logger, req ctrl.Request, k k8s.Client, now time.Time) error {
	ll := log.WithValues("lokistack", req.NamespacedName, "event", "schemaUpgrade")

	var stack lokiv1.LokiStack
	if err := k.Get(ctx, req.NamespacedName, &stack); err != nil {
		if apierrors.IsNotFound(err) {
			// maybe the user deleted it before we could react? Either way this isn't an issue
			ll.Error(err, "could not find the requested loki stack", "name", req.NamespacedName)
			return nil
		}
		return kverrors.Wrap(err, "failed to lookup lokistack", "name", req.NamespacedName)
	}

	if !stack.Spec.Storage.AllowAutomaticUpgrade {
		// automatic upgrade not enabled -> skip
		ll.Info("Skip: Automatic upgrade not enabled")
		return nil
	}

	proposedUpgrade := stack.Status.Storage.AutomaticUpgrade
	if proposedUpgrade == nil {
		// we do not have any pending upgrade
		ll.Info("Skip: No automatic upgrade scheduled")
		return nil
	}

	if now.Before(proposedUpgrade.UpgradeTime.Time) {
		// it's not time yet
		ll.Info("Skip: Scheduled automatic upgrade time not reached yet")
		return nil
	}

	readyIdx := -1
	for i, c := range stack.Status.Conditions {
		if c.Type == string(lokiv1.ConditionReady) {
			readyIdx = i
			break
		}
	}

	if readyIdx == -1 || stack.Status.Conditions[readyIdx].Status != metav1.ConditionTrue {
		// there is something else going on with the Stack, don't run the upgrade
		ll.Info("Skip: LokiStack does not have active Ready condition")
		return nil
	}

	ll.Info("Running automatic upgrade of storage schemas")
	specUpdater := func(stack *lokiv1.LokiStack) *lokiv1.LokiStack {
		modified := stack.DeepCopy()
		modified.Spec.Storage.Schemas = proposedUpgrade.Schemas
		modified.Status.Storage.AutomaticUpgrade = nil
		return modified
	}

	modified := specUpdater(&stack)
	err := k.Update(ctx, modified)
	switch {
	case err == nil:
		return nil
	case apierrors.IsConflict(err):
		// break into retry-logic below on conflict
		break
	case err != nil:
		// return non-conflict errors
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := k.Get(ctx, req.NamespacedName, &stack); err != nil {
			return err
		}

		ll.Info("Retrying automatic upgrade of storage schemas")
		modified := specUpdater(&stack)
		return k.Update(ctx, modified)
	})
}
