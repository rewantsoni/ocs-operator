package storagecluster

import (
	"fmt"
	"strings"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v4/v1"
	"github.com/red-hat-storage/ocs-operator/v4/controllers/util"
	storagev1 "k8s.io/api/storage/v1"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
)

func generateNameForCephCluster(initData *ocsv1.StorageCluster) string {
	return generateNameForCephClusterFromString(initData.Name)
}

func generateNameForCephClusterFromString(name string) string {
	return fmt.Sprintf("%s-cephcluster", name)
}

func generateNameForCephNFS(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephnfs", initData.Name)
}

func generateNameForNFSService(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-service", generateNameForCephNFS(initData))
}

func generateNameForNFSServiceMonitor(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-servicemonitor", generateNameForCephNFS(initData))
}

func generateNameForCephNFSBlockPool(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-builtin-pool", generateNameForCephNFS(initData))
}

func generateNameForCephObjectStoreUser(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephobjectstoreuser", initData.Name)
}

func generateNameForCephObjectStore(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-%s", initData.Name, "cephobjectstore")
}

func generateNameForCephRgwSC(initData *ocsv1.StorageCluster) string {
	if initData.Spec.ManagedResources.CephObjectStores.StorageClassName != "" {
		return initData.Spec.ManagedResources.CephObjectStores.StorageClassName
	}
	return fmt.Sprintf("%s-ceph-rgw", initData.Name)
}

func (r *StorageClusterReconciler) generateNameForExternalModeCephBlockPoolSC(nb *nbv1.NooBaa) (string, error) {
	var storageClassName string

	if nb.Spec.DBStorageClass != nil {
		storageClassName = *nb.Spec.DBStorageClass
	} else {
		// list all storage classes and pick the first one
		// whose provisioner suffix matches with `rbd.csi.ceph.com` suffix
		storageClasses := &storagev1.StorageClassList{}
		err := r.Client.List(r.ctx, storageClasses)
		if err != nil {
			r.Log.Error(err, "Failed to list storage classes")
			return "", err
		}

		for _, sc := range storageClasses.Items {
			if strings.HasSuffix(sc.Provisioner, "rbd.csi.ceph.com") {
				storageClassName = sc.Name
				break
			}
		}
	}

	if storageClassName == "" {
		return "", fmt.Errorf("no storage class found with provisioner suffix `rbd.csi.ceph.com`")
	}

	return storageClassName, nil
}

func generateNameForEncryptedCephBlockPoolSC(initData *ocsv1.StorageCluster) string {
	if initData.Spec.Encryption.StorageClassName != "" {
		return initData.Spec.Encryption.StorageClassName
	}
	return fmt.Sprintf("%s-ceph-rbd-encrypted", initData.Name)
}

func generateNameForCephNetworkFilesystemSC(initData *ocsv1.StorageCluster) string {
	if initData.Spec.NFS.StorageClassName != "" {
		return initData.Spec.NFS.StorageClassName
	}
	return fmt.Sprintf("%s-ceph-nfs", initData.Name)
}

func generateNameForSnapshotClassDriver(snapshotType util.SnapshotterType) string {
	return fmt.Sprintf("%s.%s.csi.ceph.com", util.StorageClassDriverNamePrefix, snapshotType)
}

func setParameterBasedOnSnapshotterType(instance *ocsv1.StorageCluster, groupSnapshotType util.GroupSnapshotterType) (string, string) {
	if groupSnapshotType == util.RbdGroupSnapshotter {
		return "pool", util.GenerateNameForCephBlockPool(instance.Name)
	}
	return "fsName", util.GenerateNameForCephFilesystem(instance.Name)
}

func generateNameForSnapshotClassSecret(instance *ocsv1.StorageCluster, snapshotType util.SnapshotterType) string {
	// nfs uses the same cephfs secrets
	if snapshotType == "nfs" {
		snapshotType = "cephfs"
	}
	if instance.Spec.ExternalStorage.Enable {
		data, ok := externalOCSResources[instance.UID]
		if !ok {
			log.Error(fmt.Errorf("Unable to retrieve external resource from externalOCSResources"),
				"unable to generate name for snapshot class secret for external mode")
		}
		// print the Secret name which contains the prefix as the rook-csi-rbd/cephfs-provisioner default secret name
		// for example if the secret name is rook-csi-rbd-node-rookStorage-replicapool it will check the prefix with rook-csi-rbd-node if it matches it will return that name
		for _, d := range data {
			if d.Kind == "Secret" {
				if strings.Contains(d.Name, fmt.Sprintf("rook-csi-%s-provisioner", snapshotType)) {
					return d.Name
				}
			}
		}
	}
	return fmt.Sprintf("rook-csi-%s-provisioner", snapshotType)
}

func generateNameForCephRbdMirror(initData *ocsv1.StorageCluster) string {
	return fmt.Sprintf("%s-cephrbdmirror", initData.Name)
}

// generateCephReplicatedSpec returns the ReplicatedSpec for the cephCluster
// based on the StorageCluster configuration
func generateCephReplicatedSpec(initData *ocsv1.StorageCluster, poolType string) cephv1.ReplicatedSpec {
	crs := cephv1.ReplicatedSpec{}

	crs.Size = getCephPoolReplicatedSize(initData)
	crs.ReplicasPerFailureDomain = uint(getReplicasPerFailureDomain(initData))
	if poolType == poolTypeData {
		crs.TargetSizeRatio = .49
	}

	return crs
}

// generateStorageQuotaName function generates a name for ClusterResourceQuota
func generateStorageQuotaName(storageClassName, quotaName string) string {
	return fmt.Sprintf("%s-%s", storageClassName, quotaName)
}
