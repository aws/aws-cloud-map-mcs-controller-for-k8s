package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/api/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// TODO move to configuration
	syncPeriod = 2 * time.Second
)

// CloudMapReconciler reconciles state of Cloud Map services with local ServiceImport objects
type CloudMapReconciler struct {
	Client   client.Client
	Cloudmap cloudmap.ServiceDiscoveryClient
	Log      common.Logger
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=create;get;list;watch;update;delete
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=list;get;create;watch;update;delete
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceimports,verbs=create;get;list;watch;update;patch;delete

// Start implements manager.Runnable
func (r *CloudMapReconciler) Start(ctx context.Context) error {
	ticker := time.NewTicker(syncPeriod)
	defer ticker.Stop()
	for {
		if err := r.Reconcile(ctx); err != nil {
			// just log the error and continue running
			r.Log.Error(err, "Cloud Map reconciliation error")
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			r.Log.Info("terminating CloudMapReconciler")
			return nil
		}
	}
}

// Reconcile triggers a single reconciliation round
func (r *CloudMapReconciler) Reconcile(ctx context.Context) error {
	namespaces := v1.NamespaceList{}
	if err := r.Client.List(ctx, &namespaces); err != nil {
		r.Log.Error(err, "unable to list namespaces")
		return err
	}

	//TODO: Fetch list of namespaces from Cloudmap and only reconcile the intersection

	for _, ns := range namespaces.Items {
		if err := r.reconcileNamespace(ctx, ns.Name); err != nil {
			return err
		}
	}

	return nil
}

func (r *CloudMapReconciler) reconcileNamespace(ctx context.Context, namespaceName string) error {
	r.Log.Debug("syncing namespace", "namespace", namespaceName)

	desiredServices, err := r.Cloudmap.ListServices(ctx, namespaceName)
	if err != nil {
		return err
	}

	serviceImports := v1alpha1.ServiceImportList{}
	if err = r.Client.List(ctx, &serviceImports, client.InNamespace(namespaceName)); err != nil {
		r.Log.Error(err, "failed to reconcile namespace", "namespace", namespaceName)
		return nil
	}

	existingImportsMap := make(map[string]v1alpha1.ServiceImport)
	for _, svc := range serviceImports.Items {
		existingImportsMap[svc.Namespace+"/"+svc.Name] = svc
	}

	for _, svc := range desiredServices {
		if len(svc.Endpoints) == 0 {
			// skip empty services
			continue
		}

		if err = r.reconcileService(ctx, svc); err != nil {
			r.Log.Error(err, "error when syncing service", "namespace", svc.Namespace, "name", svc.Name)
		}
		delete(existingImportsMap, svc.Namespace+"/"+svc.Name)
	}

	// delete remaining imports that have not been matched
	for _, i := range existingImportsMap {
		if err = r.Client.Delete(ctx, &i); err != nil {
			r.Log.Error(err, "error deleting ServiceImport", "namespace", i.Namespace, "name", i.Name)
			continue
		}
		r.Log.Info("delete ServiceImport", "namespace", i.Namespace, "name", i.Name)
	}

	return nil
}

func (r *CloudMapReconciler) reconcileService(ctx context.Context, svc *model.Service) error {
	r.Log.Info("syncing service", "namespace", svc.Namespace, "service", svc.Name)

	importedSvcPorts := ExtractServicePorts(svc.Endpoints)

	svcImport, err := r.getServiceImport(ctx, svc.Namespace, svc.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// create ServiceImport if it doesn't exist
		if svcImport, err = r.createAndGetServiceImport(ctx, svc.Namespace, svc.Name, importedSvcPorts); err != nil {
			return err
		}
	}

	derivedService, err := r.getDerivedService(ctx, svc.Namespace, svcImport.Annotations[DerivedServiceAnnotation])
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// create derived Service if it doesn't exist
		if derivedService, err = r.createAndGetDerivedService(ctx, svcImport, importedSvcPorts); err != nil {
			return err
		}
	}

	// update service import to match derived service cluster IP and imported ports if necessary
	if err = r.updateServiceImport(ctx, svcImport, derivedService, importedSvcPorts); err != nil {
		return err
	}

	// update derived service ports to match imported ports if necessary
	if err = r.updateDerivedService(ctx, derivedService, importedSvcPorts); err != nil {
		return err
	}

	return r.updateEndpointSlices(ctx, svcImport, svc.Endpoints, derivedService)
}

func (r *CloudMapReconciler) getServiceImport(ctx context.Context, namespace string, name string) (*v1alpha1.ServiceImport, error) {
	existingServiceImport := &v1alpha1.ServiceImport{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, existingServiceImport)
	return existingServiceImport, err
}

func (r *CloudMapReconciler) createAndGetServiceImport(ctx context.Context, namespace string, name string, servicePorts []*model.Port) (*v1alpha1.ServiceImport, error) {
	toCreate := CreateServiceImportStruct(namespace, name, servicePorts)
	if err := r.Client.Create(ctx, toCreate); err != nil {
		return nil, err
	}
	r.Log.Info("created ServiceImport", "namespace", namespace, "name", name)

	return r.getServiceImport(ctx, namespace, name)
}

func (r *CloudMapReconciler) getDerivedService(ctx context.Context, namespace string, name string) (*v1.Service, error) {
	existingService := &v1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, existingService)
	return existingService, err
}

func (r *CloudMapReconciler) createAndGetDerivedService(ctx context.Context, svcImport *v1alpha1.ServiceImport, svcPorts []*model.Port) (*v1.Service, error) {
	toCreate := CreateDerivedServiceStruct(svcImport, svcPorts)
	if err := r.Client.Create(ctx, toCreate); err != nil {
		return nil, err
	}
	r.Log.Info("created derived Service", "namespace", toCreate.Namespace, "name", toCreate.Name)

	return r.getDerivedService(ctx, toCreate.Namespace, svcImport.Annotations[DerivedServiceAnnotation])
}

func (r *CloudMapReconciler) updateEndpointSlices(ctx context.Context, svcImport *v1alpha1.ServiceImport, desiredEndpoints []*model.Endpoint, svc *v1.Service) error {
	existingSlicesList := discovery.EndpointSliceList{}
	if err := r.Client.List(ctx, &existingSlicesList,
		client.InNamespace(svc.Namespace), client.MatchingLabels{discovery.LabelServiceName: svc.Name}); err != nil {
		return err
	}

	plan := EndpointSlicePlan{
		Current:           existingSlicesList.Items,
		Desired:           desiredEndpoints,
		Service:           svc,
		ServiceImportName: svcImport.Name,
	}

	changes := plan.CalculateChanges()

	for _, sliceToUpdate := range changes.Update {
		r.Log.Debug("updating EndpointSlice", "namespace", sliceToUpdate.Namespace, "name", sliceToUpdate.Name)
		if err := r.Client.Update(ctx, sliceToUpdate); err != nil {
			return fmt.Errorf("failed to update EndpointSlice: %w", err)
		}
	}

	for _, sliceToDelete := range changes.Delete {
		r.Log.Debug("deleting EndpointSlice", "namespace", sliceToDelete.Namespace, "name", sliceToDelete.Name)
		if err := r.Client.Delete(ctx, sliceToDelete); err != nil {
			return fmt.Errorf("failed to delete EndpointSlice: %w", err)
		}
	}

	for _, sliceToCreate := range changes.Create {
		r.Log.Debug("creating EndpointSlice", "namespace", sliceToCreate.Namespace)
		if err := r.Client.Create(ctx, sliceToCreate); err != nil {
			return fmt.Errorf("failed to create EndpointSlice: %w", err)
		}
	}

	return nil
}

func (r *CloudMapReconciler) updateServiceImport(ctx context.Context, svcImport *v1alpha1.ServiceImport, svc *v1.Service, importedSvcPorts []*model.Port) error {
	updateRequired := false
	if len(svcImport.Spec.IPs) != 1 || svcImport.Spec.IPs[0] != svc.Spec.ClusterIP {
		r.Log.Debug("ServiceImport IP need update", "ServiceImport IPs", svcImport.Spec.IPs, "cluster IP", svc.Spec.ClusterIP)
		svcImport.Spec.IPs = []string{svc.Spec.ClusterIP}
		updateRequired = true
	}

	// ServiceImport ports do not have TargetPort, exclude field for purpose of comparison
	simplifiedSvcPorts := make([]*model.Port, 0)
	for _, svcPort := range importedSvcPorts {
		simplifiedSvcPorts = append(simplifiedSvcPorts, &model.Port{
			Name:     svcPort.Name,
			Port:     svcPort.Port,
			Protocol: svcPort.Protocol,
		})
	}

	svcImportPorts := make([]*model.Port, 0)
	for _, importPort := range svcImport.Spec.Ports {
		port := ServiceImportPortToPort(importPort)
		svcImportPorts = append(svcImportPorts, &port)
	}

	if !PortsEqualIgnoreOrder(svcImportPorts, simplifiedSvcPorts) {
		r.Log.Debug("ServiceImport ports need update", "ServiceImport Ports", svcImport.Spec.Ports, "imported ports", importedSvcPorts)
		serviceImportPorts := make([]v1alpha1.ServicePort, 0)
		for _, port := range importedSvcPorts {
			serviceImportPorts = append(serviceImportPorts, PortToServiceImportPort(*port))
		}
		svcImport.Spec.Ports = serviceImportPorts
		updateRequired = true
	}

	if updateRequired {
		if err := r.Client.Update(ctx, svcImport); err != nil {
			return err
		}
		r.Log.Info("updated ServiceImport",
			"namespace", svcImport.Namespace, "name", svcImport.Name,
			"IP", svcImport.Spec.IPs, "ports", svcImport.Spec.Ports)
	}

	return nil
}

func (r *CloudMapReconciler) updateDerivedService(ctx context.Context, svc *v1.Service, importedSvcPorts []*model.Port) error {
	svcPorts := make([]*model.Port, 0)
	for _, p := range svc.Spec.Ports {
		port := ServicePortToPort(p)
		svcPorts = append(svcPorts, &port)
	}

	portsMatch := PortsEqualIgnoreOrder(importedSvcPorts, svcPorts)
	if !portsMatch {
		newSvcPorts := make([]v1.ServicePort, 0)
		for _, importPort := range importedSvcPorts {
			newSvcPorts = append(newSvcPorts, PortToServicePort(*importPort))
		}

		svc.Spec.Ports = newSvcPorts
		if err := r.Client.Update(ctx, svc); err != nil {
			return err
		}
		r.Log.Info("updated derived Service",
			"namespace", svc.Namespace, "name", svc.Name, "ports", svc.Spec.Ports)
	}

	return nil
}
