package crd

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/red-hat-storage/ocs-operator/v4/controllers/util"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/klog/v2"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CustomResourceDefinitionReconciler reconciles a CustomResourceDefinition object
// nolint:revive
type CustomResourceDefinitionReconciler struct {
	Client        client.Client
	ctx           context.Context
	Log           logr.Logger
	AvailableCrds map[string]bool
}

// Reconcile compares available CRDs maps following either a Create or Delete event
func (r *CustomResourceDefinitionReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.ctx = ctx
	r.Log.Info("Reconciling CustomResourceDefinition.", "CRD", klog.KRef(request.Namespace, request.Name))

	var err error
	availableCrds, err := util.MapCRDAvailability(ctx, r.Client, util.CRDList...)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !reflect.DeepEqual(availableCrds, r.AvailableCrds) {
		r.Log.Info("CustomResourceDefinitions created/deleted. Restarting process.")
		panic("CustomResourceDefinitions created/deleted. Restarting process.")
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up a controller with a manager
func (r *CustomResourceDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	crdPredicate := predicate.Funcs{
		CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
			crdAvailable, keyExist := r.AvailableCrds[e.Object.GetName()]
			if keyExist && !crdAvailable {
				r.Log.Info("CustomResourceDefinition %s was Created.", e.Object.GetName())
				return true
			}
			return false
		},
		DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
			crdAvailable, keyExist := r.AvailableCrds[e.Object.GetName()]
			if keyExist && crdAvailable {
				r.Log.Info("CustomResourceDefinition %s was Deleted.", e.Object.GetName())
				return true
			}
			return false
		},
		UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
			return false
		},
		GenericFunc: func(e event.TypedGenericEvent[client.Object]) bool {
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiextensionsv1.CustomResourceDefinition{}, builder.WithPredicates(crdPredicate)).
		Complete(r)
}
