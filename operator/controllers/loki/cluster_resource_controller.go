package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	configv1 "github.com/grafana/loki/operator/apis/config/v1"
	"github.com/grafana/loki/operator/controllers/loki/internal/lokistack"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ClusterResourceReconciler reconciles annotations on all
// known LokiStack objects when any of the watched cluster-scoped
// resources receives a create/update/delete event.
type ClusterResourceReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	FeatureGates configv1.FeatureGates
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := lokistack.AnnotateForRequiredClusterResource(ctx, r.Client)
	if err != nil {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Second,
		}, err
	}
	return ctrl.Result{}, nil
}

func (r *ClusterResourceReconciler) SetupWithManager(mgr manager.Manager) error {
	return r.buildController(ctrl.NewControllerManagedBy(mgr))
}

func (r *ClusterResourceReconciler) buildController(bld *builder.Builder) error {
	hasFor := false
	if r.FeatureGates.OpenShift.ClusterTLSPolicy {
		hasFor = true
		bld = bld.For(&openshiftconfigv1.APIServer{}, builder.OnlyMetadata)
	}
	if r.FeatureGates.OpenShift.ClusterProxy {
		if hasFor {
			bld = bld.Watches(
				&source.Kind{Type: &openshiftconfigv1.Proxy{}},
				&handler.EnqueueRequestForObject{},
				builder.OnlyMetadata,
			)
		} else {
			bld = bld.For(&openshiftconfigv1.Proxy{}, builder.OnlyMetadata)
		}
	}
	return bld.Complete(r)
}
