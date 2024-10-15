package server

import (
	"context"
	ramenv1alpha1 "github.com/ramendr/ramen/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type maintenanceModeManager struct {
	client    client.Client
	namespace string
}

func newMaintenanceModeManager(cl client.Client, namespace string) (*maintenanceModeManager, error) {
	return &maintenanceModeManager{
		client:    cl,
		namespace: namespace,
	}, nil
}

func (m maintenanceModeManager) Create(ctx context.Context, name string, spec *ramenv1alpha1.MaintenanceModeSpec) error {
	mModeObj := &ramenv1alpha1.MaintenanceMode{}
	mModeObj.Name = name
	_, err := controllerutil.CreateOrUpdate(ctx, m.client, mModeObj, func() error {
		mModeObj.Spec = *spec
		return nil
	})
	return err
}

func (m maintenanceModeManager) Get(ctx context.Context, name string) (*ramenv1alpha1.MaintenanceMode, error) {
	mModeObj := &ramenv1alpha1.MaintenanceMode{}
	mModeObj.Name = name
	err := m.client.Get(ctx, client.ObjectKey{Name: name}, mModeObj)
	if err != nil {
		return nil, err
	}
	return mModeObj, nil
}

func (m maintenanceModeManager) Delete(ctx context.Context, name string) error {
	mModeObj := &ramenv1alpha1.MaintenanceMode{}
	mModeObj.Name = name
	err := m.client.Delete(ctx, mModeObj)
	if err != nil {
		return err
	}
	return nil
}
