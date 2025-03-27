package util

import (
	groupsnapapi "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumegroupsnapshot/v1beta1"
	snapapi "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GroupSnapshotterType string

const (
	RbdGroupSnapshotter    GroupSnapshotterType = "rbd"
	CephfsGroupSnapshotter GroupSnapshotterType = "cephfs"
)

func NewDefaultRbdGroupSnapshotClass(
	clusterID,
	provisionerSecret,
	namespace,
	pool,
	drStorageID string,
) *groupsnapapi.VolumeGroupSnapshotClass {

	gsc := &groupsnapapi.VolumeGroupSnapshotClass{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		Driver: RbdDriverName,
		Parameters: map[string]string{
			"clusterID": clusterID,
			"csi.storage.k8s.io/group-snapshotter-secret-name":      provisionerSecret,
			"csi.storage.k8s.io/group-snapshotter-secret-namespace": namespace,
			"pool": pool,
		},
		DeletionPolicy: snapapi.VolumeSnapshotContentDelete,
	}
	if drStorageID != "" {
		AddLabel(gsc, ramenDRStorageIDLabelKey, drStorageID)
	}
	return gsc
}

func NewDefaultCephFsGroupSnapshotClass(
	clusterID,
	provisionerSecret,
	namespace,
	fsName,
	drStorageID string,
) *groupsnapapi.VolumeGroupSnapshotClass {

	gsc := &groupsnapapi.VolumeGroupSnapshotClass{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		Driver: CephFSDriverName,
		Parameters: map[string]string{
			"clusterID": clusterID,
			"csi.storage.k8s.io/group-snapshotter-secret-name":      provisionerSecret,
			"csi.storage.k8s.io/group-snapshotter-secret-namespace": namespace,
			"fsName": fsName,
		},
		DeletionPolicy: snapapi.VolumeSnapshotContentDelete,
	}
	if drStorageID != "" {
		AddLabel(gsc, ramenDRStorageIDLabelKey, drStorageID)
	}
	return gsc
}
