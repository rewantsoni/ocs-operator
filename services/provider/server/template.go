package server

import (
	snapapi "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v4/v1"
	pb "github.com/red-hat-storage/ocs-operator/services/provider/api/v4"
	"github.com/red-hat-storage/ocs-operator/v4/controllers/util"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// Have a single function to generate the storageClass,
// take a unique parameter

func GenerateDefaultRbdStorageClass(clusterID, blockpoolname, provsionerSecet, nodeSecret, namespace string, isDefaultStorageClass bool, isEncrypted, SupportKetRotation, DrStorageID string) *storagev1.StorageClass {

	sc:=&storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"description": "Provides RWO Filesystem volumes, and RWO and RWX Block volumes",
				"reclaimspace.csiaddons.openshift.io/schedule": "@weekly",
			},
		},
		ReclaimPolicy:        ptr.To(corev1.PersistentVolumeReclaimDelete),
		AllowVolumeExpansion: ptr.To(true),
		Provisioner:          util.RbdDriverName,
		Parameters: map[string]string{
			"clusterID":                 clusterID,
			"pool":                      blockpoolname,
			"imageFeatures":             "layering,deep-flatten,exclusive-lock,object-map,fast-diff",
			"csi.storage.k8s.io/fstype": "ext4",
			"imageFormat":               "2",
			"csi.storage.k8s.io/provisioner-secret-name":            provsionerSecet,
			"csi.storage.k8s.io/node-stage-secret-name":             nodeSecret,
			"csi.storage.k8s.io/controller-expand-secret-name":      provsionerSecet,
			"csi.storage.k8s.io/provisioner-secret-namespace":       namespace,
			"csi.storage.k8s.io/node-stage-secret-namespace":        namespace,
			"csi.storage.k8s.io/controller-expand-secret-namespace": namespace,
		},
	}

	if isDefaultStorageClass {

	}
	if isDefaultStorageClass

	return sc
}
func GenerateVirtRbdStorageClass()
func GenerateNonResilitRbdStorageClass()
...


func getStorageClassTemplates(
	storageCluster *ocsv1.StorageCluster,
	consumerConfig util.StorageConsumerResources,
	fsid string,
	clientNamespace string,
) map[string]*pb.ExternalResource {

	// SID for RamenDR
	rbdStorageID := calculateCephRbdStorageID(
		fsid,
		consumerConfig.GetRbdRadosNamespaceName(),
	)

	// SID for RamenDR
	cephFsStorageID := calculateCephFsStorageID(
		fsid,
		consumerConfig.GetSubVolumeGroupName(),
	)

	return map[string]*pb.ExternalResource{
		util.GenerateNameForCephBlockPoolSC(storageCluster): {
			Name: util.GenerateNameForCephBlockPoolSC(storageCluster),
			Kind: "StorageClass",
			Data: mustMarshal(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.GenerateNameForCephBlockPoolSC(storageCluster),
					Namespace: clientNamespace,
					Annotations: map[string]string{
						"description": "Provides RWO Filesystem volumes, and RWO and RWX Block volumes",
						"reclaimspace.csiaddons.openshift.io/schedule": "@weekly",
					},
					Labels: map[string]string{
						ramenDRStorageIDLabelKey: rbdStorageID,
					},
				},
				ReclaimPolicy:        ptr.To(corev1.PersistentVolumeReclaimDelete),
				AllowVolumeExpansion: ptr.To(true),
				Provisioner:          util.RbdDriverName,
				Parameters: map[string]string{
					"clusterID":                 consumerConfig.GetRbdClientProfileName(),
					"pool":                      util.GenerateNameForCephBlockPool(storageCluster.Name),
					"imageFeatures":             "layering,deep-flatten,exclusive-lock,object-map,fast-diff",
					"csi.storage.k8s.io/fstype": "ext4",
					"imageFormat":               "2",
					"csi.storage.k8s.io/provisioner-secret-name":            consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/node-stage-secret-name":             consumerConfig.GetCsiRbdNodeSecretName(),
					"csi.storage.k8s.io/controller-expand-secret-name":      consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/provisioner-secret-namespace":       clientNamespace,
					"csi.storage.k8s.io/node-stage-secret-namespace":        clientNamespace,
					"csi.storage.k8s.io/controller-expand-secret-namespace": clientNamespace,
				},
			}),
		},
		util.GenerateNameForNonResilientCephBlockPoolSC(storageCluster): {
			Name: util.GenerateNameForNonResilientCephBlockPoolSC(storageCluster),
			Kind: "StorageClass",
			Data: mustMarshal(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.GenerateNameForNonResilientCephBlockPoolSC(storageCluster),
					Namespace: clientNamespace,
					Annotations: map[string]string{
						"description": "Ceph Non Resilient Pools : Provides RWO Filesystem volumes, and RWO and RWX Block volumes",
						"reclaimspace.csiaddons.openshift.io/schedule": "@weekly",
					},
					Labels: map[string]string{
						ramenDRStorageIDLabelKey: rbdStorageID,
					},
				},
				ReclaimPolicy:        ptr.To(corev1.PersistentVolumeReclaimDelete),
				AllowVolumeExpansion: ptr.To(true),
				Provisioner:          util.RbdDriverName,
				Parameters: map[string]string{
					"clusterID":                 consumerConfig.GetRbdClientProfileName(),
					"topologyConstrainedPools":  util.GetTopologyConstrainedPools(storageCluster),
					"imageFeatures":             "layering,deep-flatten,exclusive-lock,object-map,fast-diff",
					"csi.storage.k8s.io/fstype": "ext4",
					"imageFormat":               "2",
					"csi.storage.k8s.io/provisioner-secret-name":            consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/node-stage-secret-name":             consumerConfig.GetCsiRbdNodeSecretName(),
					"csi.storage.k8s.io/controller-expand-secret-name":      consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/provisioner-secret-namespace":       clientNamespace,
					"csi.storage.k8s.io/node-stage-secret-namespace":        clientNamespace,
					"csi.storage.k8s.io/controller-expand-secret-namespace": clientNamespace,
				},
			}),
		},
		util.GenerateNameForCephBlockPoolVirtualizationSC(storageCluster): {
			Name: util.GenerateNameForCephBlockPoolVirtualizationSC(storageCluster),
			Kind: "StorageClass",
			Data: mustMarshal(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.GenerateNameForCephBlockPoolVirtualizationSC(storageCluster),
					Namespace: clientNamespace,
					Annotations: map[string]string{
						"description": "Provides RWO and RWX Block volumes suitable for Virtual Machine disks",
						"reclaimspace.csiaddons.openshift.io/schedule":   "@weekly",
						"storageclass.kubevirt.io/is-default-virt-class": "true",
					},
					Labels: map[string]string{
						ramenDRStorageIDLabelKey: rbdStorageID,
					},
				},
				ReclaimPolicy:        ptr.To(corev1.PersistentVolumeReclaimDelete),
				AllowVolumeExpansion: ptr.To(true),
				Provisioner:          util.RbdDriverName,
				Parameters: map[string]string{
					"clusterID":                 consumerConfig.GetRbdClientProfileName(),
					"pool":                      util.GenerateNameForCephBlockPool(storageCluster.Name),
					"imageFeatures":             "layering,deep-flatten,exclusive-lock,object-map,fast-diff",
					"csi.storage.k8s.io/fstype": "ext4",
					"imageFormat":               "2",
					"csi.storage.k8s.io/provisioner-secret-name":            consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/node-stage-secret-name":             consumerConfig.GetCsiRbdNodeSecretName(),
					"csi.storage.k8s.io/controller-expand-secret-name":      consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/provisioner-secret-namespace":       clientNamespace,
					"csi.storage.k8s.io/node-stage-secret-namespace":        clientNamespace,
					"csi.storage.k8s.io/controller-expand-secret-namespace": clientNamespace,
				},
			}),
		},
		util.GenerateNameForCephFilesystemSC(storageCluster): {
			Name: util.GenerateNameForCephFilesystemSC(storageCluster),
			Kind: "StorageClass",
			Data: mustMarshal(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.GenerateNameForCephFilesystemSC(storageCluster),
					Namespace: clientNamespace,
					Annotations: map[string]string{
						"description": "Provides RWO and RWX Filesystem volumes",
					},
					Labels: map[string]string{
						ramenDRStorageIDLabelKey: cephFsStorageID,
					},
				},
				ReclaimPolicy:        ptr.To(corev1.PersistentVolumeReclaimDelete),
				AllowVolumeExpansion: ptr.To(true),
				Provisioner:          util.RbdDriverName,
				Parameters: map[string]string{
					"clusterID":                 consumerConfig.GetCephFsClientProfileName(),
					"subvolumegroupname":        consumerConfig.GetSubVolumeGroupName(),
					"fsName":                    util.GenerateNameForCephFilesystem(storageCluster.Name),
					"imageFeatures":             "layering,deep-flatten,exclusive-lock,object-map,fast-diff",
					"csi.storage.k8s.io/fstype": "ext4",
					"imageFormat":               "2",
					"csi.storage.k8s.io/provisioner-secret-name":            consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/node-stage-secret-name":             consumerConfig.GetCsiRbdNodeSecretName(),
					"csi.storage.k8s.io/controller-expand-secret-name":      consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/provisioner-secret-namespace":       clientNamespace,
					"csi.storage.k8s.io/node-stage-secret-namespace":        clientNamespace,
					"csi.storage.k8s.io/controller-expand-secret-namespace": clientNamespace,
				},
			}),
		},
	}
}

func getVolumeSnapshotClassTemplates(
	storageCluster *ocsv1.StorageCluster,
	consumerConfig util.StorageConsumerResources,
	fsid string,
	clientNamespace string,
) map[string]*pb.ExternalResource {

	// SID for RamenDR
	rbdStorageID := calculateCephRbdStorageID(
		fsid,
		consumerConfig.GetRbdRadosNamespaceName(),
	)

	// SID for RamenDR
	cephFsStorageID := calculateCephFsStorageID(
		fsid,
		consumerConfig.GetSubVolumeGroupName(),
	)

	return map[string]*pb.ExternalResource{
		util.GenerateNameForSnapshotClass(storageCluster, util.RbdSnapshotter): {
			Name: util.GenerateNameForSnapshotClass(storageCluster, util.RbdSnapshotter),
			Kind: "VolumeSnapshotClass",
			Data: mustMarshal(&snapapi.VolumeSnapshotClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: util.GenerateNameForSnapshotClass(storageCluster, util.RbdSnapshotter),
					Labels: map[string]string{
						ramenDRStorageIDLabelKey: rbdStorageID,
					},
				},
				Driver: util.RbdDriverName,
				Parameters: map[string]string{
					"clusterID": consumerConfig.GetRbdClientProfileName(),
					"csi.storage.k8s.io/snapshotter-secret-name":      consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/snapshotter-secret-namespace": clientNamespace,
				},
				DeletionPolicy: snapapi.VolumeSnapshotContentDelete,
			}),
		},
		util.GenerateNameForSnapshotClass(storageCluster, util.CephfsSnapshotter): {
			Name: util.GenerateNameForSnapshotClass(storageCluster, util.CephfsSnapshotter),
			Kind: "VolumeSnapshotClass",
			Data: mustMarshal(&snapapi.VolumeSnapshotClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: util.GenerateNameForSnapshotClass(storageCluster, util.CephfsSnapshotter),
					Labels: map[string]string{
						ramenDRStorageIDLabelKey: cephFsStorageID,
					},
				},
				Driver: util.RbdDriverName,
				Parameters: map[string]string{
					"clusterID": consumerConfig.GetCephFsClientProfileName(),
					"csi.storage.k8s.io/snapshotter-secret-name":      consumerConfig.GetCsiCephFsProvisionerSecretName(),
					"csi.storage.k8s.io/snapshotter-secret-namespace": clientNamespace,
				},
				DeletionPolicy: snapapi.VolumeSnapshotContentDelete,
			}),
		},
	}
}

func getVolumeGroupSnapshotClassTemplates(
	storageCluster *ocsv1.StorageCluster,
	consumerConfig util.StorageConsumerResources,
	fsid string,
	clientNamespace string,
) map[string]*pb.ExternalResource {

	// SID for RamenDR
	rbdStorageID := calculateCephRbdStorageID(
		fsid,
		consumerConfig.GetRbdRadosNamespaceName(),
	)

	// SID for RamenDR
	cephFsStorageID := calculateCephFsStorageID(
		fsid,
		consumerConfig.GetSubVolumeGroupName(),
	)

	return map[string]*pb.ExternalResource{
		util.GenerateNameForGroupSnapshotClass(storageCluster, util.RbdGroupSnapshotter): {
			Name: util.GenerateNameForGroupSnapshotClass(storageCluster, util.RbdGroupSnapshotter),
			Kind: "VolumeGroupSnapshotClass",
			Data: mustMarshal(&snapapi.VolumeSnapshotClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: util.GenerateNameForGroupSnapshotClass(storageCluster, util.RbdGroupSnapshotter),
					Labels: map[string]string{
						ramenDRStorageIDLabelKey: rbdStorageID,
					},
				},
				Driver: util.RbdDriverName,
				Parameters: map[string]string{
					"clusterID": consumerConfig.GetRbdClientProfileName(),
					"csi.storage.k8s.io/group-snapshotter-secret-name":      consumerConfig.GetCsiRbdProvisionerSecretName(),
					"csi.storage.k8s.io/group-snapshotter-secret-namespace": clientNamespace,
					"pool": util.GenerateNameForCephBlockPool(storageCluster.Name),
				},
				DeletionPolicy: snapapi.VolumeSnapshotContentDelete,
			}),
		},
		util.GenerateNameForGroupSnapshotClass(storageCluster, util.CephfsGroupSnapshotter): {
			Name: util.GenerateNameForGroupSnapshotClass(storageCluster, util.CephfsGroupSnapshotter),
			Kind: "VolumeGroupSnapshotClass",
			Data: mustMarshal(&snapapi.VolumeSnapshotClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: util.GenerateNameForGroupSnapshotClass(storageCluster, util.CephfsGroupSnapshotter),
					Labels: map[string]string{
						ramenDRStorageIDLabelKey: cephFsStorageID,
					},
				},
				Driver: util.RbdDriverName,
				Parameters: map[string]string{
					"clusterID": consumerConfig.GetCephFsClientProfileName(),
					"csi.storage.k8s.io/group-snapshotter-secret-name":      consumerConfig.GetCsiCephFsProvisionerSecretName(),
					"csi.storage.k8s.io/group-snapshotter-secret-namespace": clientNamespace,
					"fsName": util.GenerateNameForCephFilesystem(storageCluster.Name),
				},
				DeletionPolicy: snapapi.VolumeSnapshotContentDelete,
			}),
		},
	}
}
