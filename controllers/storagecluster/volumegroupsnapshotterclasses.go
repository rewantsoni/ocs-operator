package storagecluster

import (
	"fmt"
	"github.com/red-hat-storage/ocs-operator/v4/controllers/util"
	"reflect"

	groupsnapapi "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumegroupsnapshot/v1beta1"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v4/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ocsGroupSnapshotClass struct{}

type GroupSnapshotClassConfiguration struct {
	groupSnapshotClass *groupsnapapi.VolumeGroupSnapshotClass
	reconcileStrategy  ReconcileStrategy
}

func newCephFilesystemGroupSnapshotClassConfiguration(instance *ocsv1.StorageCluster) GroupSnapshotClassConfiguration {
	gsc := GroupSnapshotClassConfiguration{
		groupSnapshotClass: util.NewDefaultCephFsGroupSnapshotClass(
			instance.Namespace,
			"rook-csi-cephfs-provisioner",
			instance.Namespace,
			util.GenerateNameForCephFilesystem(instance.Name),
			"",
		),
		reconcileStrategy: ReconcileStrategy(instance.Spec.ManagedResources.CephFilesystems.ReconcileStrategy),
	}
	gsc.groupSnapshotClass.Name = util.GenerateNameForGroupSnapshotClass(instance, util.CephfsGroupSnapshotter)
	return gsc
}

func newCephBlockPoolGroupSnapshotClassConfiguration(instance *ocsv1.StorageCluster) GroupSnapshotClassConfiguration {
	gsc := GroupSnapshotClassConfiguration{
		groupSnapshotClass: util.NewDefaultCephFsGroupSnapshotClass(
			instance.Namespace,
			"rook-csi-rbd-provisioner",
			instance.Namespace,
			util.GenerateNameForCephBlockPool(instance.Name),
			"",
		),
		reconcileStrategy: ReconcileStrategy(instance.Spec.ManagedResources.CephBlockPools.ReconcileStrategy),
	}
	gsc.groupSnapshotClass.Name = util.GenerateNameForGroupSnapshotClass(instance, util.RbdGroupSnapshotter)
	return gsc
}

func newGroupSnapshotClassConfigurations(instance *ocsv1.StorageCluster) []GroupSnapshotClassConfiguration {
	vsccs := []GroupSnapshotClassConfiguration{
		newCephFilesystemGroupSnapshotClassConfiguration(instance),
		newCephBlockPoolGroupSnapshotClassConfiguration(instance),
	}
	return vsccs
}

func (r *StorageClusterReconciler) createGroupSnapshotClasses(vsccs []GroupSnapshotClassConfiguration) error {

	for _, vscc := range vsccs {
		if vscc.reconcileStrategy == ReconcileStrategyIgnore {
			continue
		}

		vsc := vscc.groupSnapshotClass
		existing := &groupsnapapi.VolumeGroupSnapshotClass{}
		err := r.Client.Get(r.ctx, types.NamespacedName{Name: vsc.Name, Namespace: vsc.Namespace}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				// Since the SnapshotClass is not found, we will create a new one
				r.Log.Info("Creating GroupSnapshotClass.", "GroupSnapshotClass", klog.KRef("", vsc.Name))
				err = r.Client.Create(r.ctx, vsc)
				if err != nil {
					r.Log.Error(err, "Failed to create GroupSnapshotClass.", "GroupSnapshotClass", klog.KRef("", vsc.Name))
					return err
				}
				// no error, continue with the next iteration
				continue
			}

			r.Log.Error(err, "Failed to 'Get' GroupSnapshotClass.", "GroupSnapshotClass", klog.KRef("", vsc.Name))
			return err
		}
		if vscc.reconcileStrategy == ReconcileStrategyInit {
			return nil
		}
		if existing.DeletionTimestamp != nil {
			return fmt.Errorf("failed to restore GroupSnapshotClass %q because it is marked for deletion", existing.Name)
		}
		// if there is a mismatch in the parameters of existing vs created resources,
		if !reflect.DeepEqual(vsc.Parameters, existing.Parameters) {
			// we have to update the existing SnapshotClass
			r.Log.Info("GroupSnapshotClass needs to be updated", "GroupSnapshotClass", klog.KRef("", existing.Name))
			existing.ObjectMeta.OwnerReferences = vsc.ObjectMeta.OwnerReferences
			vsc.ObjectMeta = existing.ObjectMeta
			if err := r.Client.Update(r.ctx, vsc); err != nil {
				r.Log.Error(err, "GroupSnapshotClass updation failed.", "GroupSnapshotClass", klog.KRef("", existing.Name))
				return err
			}
		}
	}
	return nil
}

func (obj *ocsGroupSnapshotClass) ensureCreated(r *StorageClusterReconciler, instance *ocsv1.StorageCluster) (reconcile.Result, error) {
	if !r.AvailableCrds[VolumeGroupSnapshotClassCrdName] {
		r.Log.Info("VolumeGroupSnapshotClass CRD is not available")
		return reconcile.Result{}, nil
	}

	vgsc := newGroupSnapshotClassConfigurations(instance)

	err := r.createGroupSnapshotClasses(vgsc)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (obj *ocsGroupSnapshotClass) ensureDeleted(r *StorageClusterReconciler, instance *ocsv1.StorageCluster) (reconcile.Result, error) {
	if !r.AvailableCrds[VolumeGroupSnapshotClassCrdName] {
		r.Log.Info("VolumeGroupSnapshotClass CRD doesn't exist")
		return reconcile.Result{}, nil
	}

	vgscs := newGroupSnapshotClassConfigurations(instance)
	for _, vgsc := range vgscs {
		sc := vgsc.groupSnapshotClass
		err := r.Client.Delete(r.ctx, sc)
		if err != nil {
			if errors.IsNotFound(err) {
				r.Log.Info("Uninstall: GroupSnapshotClass not found, nothing to do.", "GroupSnapshotClass", klog.KRef("", sc.Name))
			} else {
				r.Log.Error(err, "Uninstall: Error while deleting GroupSnapshotClass.", "GroupSnapshotClass", klog.KRef("", sc.Name))
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}
