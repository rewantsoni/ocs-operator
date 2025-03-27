package storagecluster

import (
	"fmt"
	"strings"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v4/v1"
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
