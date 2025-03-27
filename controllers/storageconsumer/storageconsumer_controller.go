/*
Copyright 2021 Red Hat OpenShift Container Storage.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"cmp"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.uber.org/multierr"
	"slices"
	"strings"

	"github.com/red-hat-storage/ocs-operator/api/v4/v1alpha1"
	ocsv1alpha1 "github.com/red-hat-storage/ocs-operator/api/v4/v1alpha1"
	"github.com/red-hat-storage/ocs-operator/v4/controllers/util"

	"github.com/go-logr/logr"
	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	rookCephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	StorageConsumerAnnotation     = "ocs.openshift.io.storageconsumer"
	StorageRequestAnnotation      = "ocs.openshift.io.storagerequest"
	StorageCephUserTypeAnnotation = "ocs.openshift.io.cephusertype"
	StorageProfileLabel           = "ocs.openshift.io/storageprofile"
	ConsumerUUIDLabel             = "ocs.openshift.io/storageconsumer-uuid"
	StorageConsumerNameLabel      = "ocs.openshift.io/storageconsumer-name"
)

// StorageConsumerReconciler reconciles a StorageConsumer object
type StorageConsumerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	ctx             context.Context
	storageConsumer *ocsv1alpha1.StorageConsumer
	namespace       string
}

//+kubebuilder:rbac:groups=ocs.openshift.io,resources=storageconsumers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocs.openshift.io,resources=storageconsumers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ocs.openshift.io,resources=storagerequests,verbs=get;list;
// +kubebuilder:rbac:groups=noobaa.io,resources=noobaaaccounts,verbs=get;list;watch;create;update;delete

// Reconcile reads that state of the cluster for a StorageConsumer object and makes changes based on the state read
// and what is in the StorageConsumer.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *StorageConsumerReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	prevLogger := r.Log
	defer func() { r.Log = prevLogger }()
	r.Log = r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.ctx = ctx
	r.namespace = request.Namespace

	r.storageConsumer = &ocsv1alpha1.StorageConsumer{}
	r.storageConsumer.Name = request.Name
	r.storageConsumer.Namespace = r.namespace

	if err := r.get(r.storageConsumer); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("No StorageConsumer resource.")
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to retrieve StorageConsumer.")
		return reconcile.Result{}, err
	}

	// Reconcile changes to the cluster
	result, reconcileError := r.reconcilePhases()

	// Apply status changes to the StorageConsumer
	statusError := r.Client.Status().Update(r.ctx, r.storageConsumer)
	if statusError != nil {
		r.Log.Info("Could not update StorageConsumer status.")
	}

	// Reconcile errors have higher priority than status update errors
	if reconcileError != nil {
		return result, reconcileError
	} else if statusError != nil {
		return result, statusError
	}

	return result, nil

}

func (r *StorageConsumerReconciler) reconcilePhases() (reconcile.Result, error) {

	if !r.storageConsumer.Spec.Enable {
		r.storageConsumer.Status.State = v1alpha1.StorageConsumerStateDisabled
		return reconcile.Result{}, nil
	}

	r.storageConsumer.Status.State = v1alpha1.StorageConsumerStateConfiguring
	r.storageConsumer.Status.CephResources = []*ocsv1alpha1.CephResourcesSpec{}

	if r.storageConsumer.GetDeletionTimestamp().IsZero() {

		consumerConfigMap := &corev1.ConfigMap{}
		consumerConfigMap.Namespace = r.namespace
		consumerConfigMap.Name = cmp.Or(
			r.storageConsumer.Spec.ResourceNameMappingConfigMap.Name,
			fmt.Sprintf("storageconsumer-%v", util.FnvHash(r.storageConsumer.Name)),
		)
		if err := r.get(consumerConfigMap); client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, err
		}

		var fsid string
		if cephCluster, err := util.GetCephClusterInNamespace(r.ctx, r.Client, r.namespace); err != nil {
			return reconcile.Result{}, err
		} else if cephCluster.Status.CephStatus == nil || cephCluster.Status.CephStatus.FSID == "" {
			return reconcile.Result{}, fmt.Errorf("waiting for Ceph FSID")
		} else {
			fsid = cephCluster.Status.CephStatus.FSID
		}

		if _, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, consumerConfigMap, func() error {
			if consumerConfigMap.Data == nil {
				consumerConfigMap.Data = map[string]string{}
			}

			defaultConsumerResourceNames := util.GetStorageConsumerDefaultResourceNames(r.storageConsumer.Name, fsid)
			for key := range defaultConsumerResourceNames {
				consumerConfigMap.Data[key] = cmp.Or(
					strings.Trim(consumerConfigMap.Data[key], " "),
					defaultConsumerResourceNames[key],
				)
			}

			// Get config map's controller reference
			controllerIndex := slices.IndexFunc(
				consumerConfigMap.OwnerReferences,
				func(ref metav1.OwnerReference) bool { return ptr.Deref(ref.Controller, false) },
			)
			var controllerRef *metav1.OwnerReference
			if controllerIndex != -1 {
				controllerRef = &consumerConfigMap.OwnerReferences[controllerIndex]
			}
			// If there is no controller ref, take control over the config map
			if controllerRef == nil {
				if err := controllerutil.SetControllerReference(
					r.storageConsumer,
					consumerConfigMap,
					r.Scheme,
				); err != nil {
					return err
				}
				// If I am not the config map controller add me as an owner
			} else if controllerRef.UID != r.storageConsumer.UID {
				if err := controllerutil.SetOwnerReference(
					r.storageConsumer,
					consumerConfigMap,
					r.Scheme,
				); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return reconcile.Result{}, err
		}

		r.storageConsumer.Status.ResourceNameMappingConfigMap = corev1.LocalObjectReference{Name: consumerConfigMap.Name}

		// Get config map's controller reference
		controllerIndex := slices.IndexFunc(
			consumerConfigMap.OwnerReferences,
			func(ref metav1.OwnerReference) bool { return ptr.Deref(ref.Controller, false) },
		)
		var controllerRef *metav1.OwnerReference
		if controllerIndex != -1 {
			controllerRef = &consumerConfigMap.OwnerReferences[controllerIndex]
		}

		isPrimaryConsumer := controllerRef != nil && controllerRef.UID == r.storageConsumer.UID

		if isPrimaryConsumer {
			consumerResources := util.WrapStorageConsumerResourceMap(consumerConfigMap.Data)

			idx := slices.IndexFunc(r.storageConsumer.OwnerReferences, func(ref metav1.OwnerReference) bool {
				return ref.Kind == "StorageCluster"
			})
			var storageCluster *metav1.OwnerReference
			if idx != -1 {
				storageCluster = &r.storageConsumer.OwnerReferences[idx]
			}
			if storageCluster == nil {
				return reconcile.Result{}, fmt.Errorf("failed to find StorgeCluster owning the StorageConsumer")
			}

			if err := r.reconcileCephRadosNamespace(
				consumerResources.GetRbdRadosNamespaceName(),
				consumerConfigMap,
			); err != nil {
				return reconcile.Result{}, err
			}

			if err := r.reconcileCephFilesystemSubVolumeGroup(
				util.GenerateNameForCephFilesystem(storageCluster.Name),
				consumerResources.GetSubVolumeGroupName(),
				consumerConfigMap,
			); err != nil {
				return reconcile.Result{}, err
			}
		}

		// A provider cluster already has a NooBaa system and does not require a NooBaa account
		// to connect to a remote cluster, unlike client clusters.
		// A NooBaa account only needs to be created if the storage consumer is for a client cluster.
		clusterID := util.GetClusterID(r.ctx, r.Client, &r.Log)
		if clusterID != "" && !strings.Contains(r.storageConsumer.Name, clusterID) {
			if err := r.reconcileNoobaaAccount(); err != nil {
				return reconcile.Result{}, err
			}
		}

		cephResourcesReady := true
		for _, cephResource := range r.storageConsumer.Status.CephResources {
			if cephResource.Phase != "Ready" {
				cephResourcesReady = false
				break
			}
		}

		if cephResourcesReady {
			r.storageConsumer.Status.State = v1alpha1.StorageConsumerStateReady
		}

	} else {
		r.storageConsumer.Status.State = v1alpha1.StorageConsumerStateDeleting
	}

	return reconcile.Result{}, nil
}

func (r *StorageConsumerReconciler) reconcileCephRadosNamespace(
	radosNamespaceName string,
	additionalOwner client.Object,
) error {
	blockPools := &rookCephv1.CephBlockPoolList{}
	if err := r.List(r.ctx, blockPools, client.InNamespace(r.namespace)); err != nil {
		return err
	}

	// ensure for this consumer a rados namespace is created in every blockpool
	var combinedErr error
	for idx := range blockPools.Items {
		bp := &blockPools.Items[idx]
		if bp.Name == "builtin-mgr" {
			continue
		}

		rns := &rookCephv1.CephBlockPoolRadosNamespace{}
		rns.Name = fmt.Sprintf("%s-%s", bp.Name, radosNamespaceName)
		rns.Namespace = r.namespace

		if _, err := ctrl.CreateOrUpdate(r.ctx, r.Client, rns, func() error {
			if err := controllerutil.SetControllerReference(r.storageConsumer, rns, r.Scheme); err != nil {
				return err
			}
			if err := controllerutil.SetOwnerReference(additionalOwner, rns, r.Scheme); err != nil {
				return err
			}
			rns.Spec.Name = radosNamespaceName
			rns.Spec.BlockPoolName = bp.Name
			return nil
		}); err != nil {
			multierr.AppendInto(&combinedErr, err)
		}
	}

	return combinedErr
}

func (r *StorageConsumerReconciler) reconcileCephFilesystemSubVolumeGroup(
	cephFileSystemName string,
	subVolumeGroupName string,
	additionalOwner client.Object,
) error {
	cephFs := &rookCephv1.CephFilesystem{}
	cephFs.Name = cephFileSystemName
	cephFs.Namespace = r.namespace
	if err := r.get(cephFs); err != nil {
		return fmt.Errorf("failed to get CephFilesystem: %v", err)
	}

	svg := &rookCephv1.CephFilesystemSubVolumeGroup{}
	svg.Name = subVolumeGroupName
	svg.Namespace = r.namespace

	if _, err := ctrl.CreateOrUpdate(r.ctx, r.Client, svg, func() error {
		if err := controllerutil.SetControllerReference(r.storageConsumer, svg, r.Scheme); err != nil {
			return err
		}
		if err := controllerutil.SetOwnerReference(additionalOwner, svg, r.Scheme); err != nil {
			return err
		}
		svg.Spec.FilesystemName = cephFs.Name
		svg.Spec.Name = subVolumeGroupName
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (r *StorageConsumerReconciler) reconcileNoobaaAccount() error {
	noobaaAccount := &nbv1.NooBaaAccount{}
	noobaaAccount.Name = r.storageConsumer.Name
	noobaaAccount.Namespace = r.storageConsumer.Namespace
	_, err := ctrl.CreateOrUpdate(r.ctx, r.Client, noobaaAccount, func() error {
		if err := r.own(noobaaAccount); err != nil {
			return err
		}
		// TODO: query the name of backing store during runtime
		noobaaAccount.Spec.DefaultResource = "noobaa-default-backing-store"
		// the following annotation will enable noobaa-operator to create a auth_token secret based on this account
		util.AddAnnotation(noobaaAccount, "remote-operator", "true")
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create noobaa account for storageConsumer %v: %v", r.storageConsumer.Name, err)
	}

	phase := string(noobaaAccount.Status.Phase)
	r.setCephResourceStatus(noobaaAccount.Name, "NooBaaAccount", phase, nil)
	return nil
}

func (r *StorageConsumerReconciler) setCephResourceStatus(name string, kind string, phase string, cephClients map[string]string) {
	cephResourceSpec := ocsv1alpha1.CephResourcesSpec{
		Name:        name,
		Kind:        kind,
		Phase:       phase,
		CephClients: cephClients,
	}
	r.storageConsumer.Status.CephResources = append(
		r.storageConsumer.Status.CephResources,
		&cephResourceSpec,
	)
}

func (r *StorageConsumerReconciler) get(obj client.Object) error {
	key := client.ObjectKeyFromObject(obj)
	return r.Client.Get(r.ctx, key, obj)
}

func (r *StorageConsumerReconciler) own(resource metav1.Object) error {
	// Ensure storageConsumer ownership on a resource
	return ctrl.SetControllerReference(r.storageConsumer, resource, r.Scheme)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StorageConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueueForAllStorageConsumers := handler.EnqueueRequestsFromMapFunc(
		func(context context.Context, obj client.Object) []reconcile.Request {
			// Get the StorageConsumer objects
			consumers := &ocsv1alpha1.StorageConsumerList{}
			err := r.Client.List(context, consumers, &client.ListOptions{Namespace: obj.GetNamespace()})
			if err != nil {
				r.Log.Error(err, "Unable to list StorageConsumers")
				return []reconcile.Request{}
			}

			// Return name and namespace of the StorageClusters object
			request := make([]reconcile.Request, len(consumers.Items))
			for i := range consumers.Items {
				request[i] = reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(&consumers.Items[i]),
				}
			}

			return request
		},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&ocsv1alpha1.StorageConsumer{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&nbv1.NooBaaAccount{}).
		Owns(&corev1.ConfigMap{}, builder.MatchEveryOwner).
		Owns(
			&rookCephv1.CephBlockPoolRadosNamespace{},
			builder.MatchEveryOwner,
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
			),
		).
		Owns(
			&rookCephv1.CephFilesystemSubVolumeGroup{},
			builder.MatchEveryOwner,
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
			),
		).
		// Watch non-owned resources
		Watches(&rookCephv1.CephBlockPool{}, enqueueForAllStorageConsumers).
		Watches(&rookCephv1.CephFilesystem{}, enqueueForAllStorageConsumers).
		Complete(r)
}

func GenerateHashForCephClient(storageConsumerName, cephUserType string) string {
	var c struct {
		StorageConsumerName string `json:"id"`
		CephUserType        string `json:"cephUserType"`
	}

	c.StorageConsumerName = storageConsumerName
	c.CephUserType = cephUserType
	cephClient, err := json.Marshal(c)

	if err != nil {
		klog.Errorf("failed to marshal ceph client name for consumer %s. %v", storageConsumerName, err)
		panic("failed to marshal")
	}
	name := md5.Sum([]byte(cephClient))
	return hex.EncodeToString(name[:16])
}
