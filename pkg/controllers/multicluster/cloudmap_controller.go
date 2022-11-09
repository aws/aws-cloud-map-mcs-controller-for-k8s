package controllers

import (
	"context"
	"fmt"
	"time"

	multiclusterv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
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
	Client       client.Client
	Cloudmap     cloudmap.ServiceDiscoveryClient
	Log          common.Logger
	ClusterUtils model.ClusterUtils
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=create;get;list;watch;update;delete
// +kubebuilder:rbac:groups=about.k8s.io,resources=clusterproperties,verbs=create;get;list;watch;update;patch;delete
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
func (r *CloudMapReconciler) Reconcile(ctx context.Context) (err error) {
	clusterProperties, err := r.ClusterUtils.GetClusterProperties(ctx)
	if err != nil {
		r.Log.Error(err, "unable to retrieve ClusterId and ClusterSetId")
		return err
	}
	r.Log.Debug("clusterProperties found", "ClusterId", clusterProperties.ClusterId(), "ClusterSetId", clusterProperties.ClusterSetId())

	namespaces := v1.NamespaceList{}
	if err = r.Client.List(ctx, &namespaces); err != nil {
		r.Log.Error(err, "unable to list cluster namespaces")
		return err
	}

	for _, ns := range namespaces.Items {
		reconErr := r.reconcileNamespace(ctx, ns.Name)
		if reconErr != nil {
			err = common.Wrap(err, reconErr)
		}
	}

	return err
}

func (r *CloudMapReconciler) reconcileNamespace(ctx context.Context, namespaceName string) (err error) {
	r.Log.Debug("syncing namespace", "namespace", namespaceName)

	desiredServices, err := r.Cloudmap.ListServices(ctx, namespaceName)
	if err != nil {
		r.Log.Error(err, "failed to fetch the list Services")
		return err
	}

	serviceImports := multiclusterv1alpha1.ServiceImportList{}
	if err = r.Client.List(ctx, &serviceImports, client.InNamespace(namespaceName)); err != nil {
		r.Log.Error(err, "failed to reconcile namespace", "namespace", namespaceName)
		return err
	}

	existingImportsMap := make(map[string]multiclusterv1alpha1.ServiceImport)
	for _, svc := range serviceImports.Items {
		existingImportsMap[svc.Name] = svc
	}

	for _, svc := range desiredServices {
		if len(svc.Endpoints) == 0 {
			// skip empty services
			continue
		}

		if reconErr := r.reconcileService(ctx, svc); reconErr != nil {
			r.Log.Error(reconErr, "error when syncing service", "namespace", svc.Namespace, "name", svc.Name)
			err = common.Wrap(err, reconErr)
		}
		delete(existingImportsMap, svc.Name)
	}

	// delete remaining imports that have not been matched
	for _, i := range existingImportsMap {
		r.Log.Info("delete ServiceImport", "namespace", i.Namespace, "name", i.Name)
		if deleteErr := r.Client.Delete(ctx, &i); deleteErr != nil {
			r.Log.Error(deleteErr, "error deleting ServiceImport", "namespace", i.Namespace, "name", i.Name)
			err = common.Wrap(err, deleteErr)
			continue
		}
	}

	return err
}

func (r *CloudMapReconciler) reconcileService(ctx context.Context, svc *model.Service) error {
	r.Log.Debug("syncing service", "namespace", svc.Namespace, "service", svc.Name)

	importedSvcPorts := ExtractServicePorts(svc.Endpoints)

	clusterIdToEndpointsMap := make(map[string][]*model.Endpoint)
	for _, ep := range svc.Endpoints {
		clusterIdToEndpointsMap[ep.ClusterId] = append(clusterIdToEndpointsMap[ep.ClusterId], ep)
	}

	clusterIds := make([]string, 0, len(clusterIdToEndpointsMap))
	for clusterId := range clusterIdToEndpointsMap {
		clusterIds = append(clusterIds, clusterId)
	}

	svcImport, err := r.getServiceImport(ctx, svc.Namespace, svc.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// create ServiceImport if it doesn't exist
		if svcImport, err = r.createAndGetServiceImport(ctx, svc, importedSvcPorts, clusterIds); err != nil {
			return err
		}
	}

	// get or create derived Service for each cluster the service is a member of
	derivedServices := make([]*v1.Service, 0, len(clusterIds))
	for _, clusterId := range clusterIds {
		endpoints := clusterIdToEndpointsMap[clusterId]
		clusterImportedSvcPorts := ExtractServicePorts(endpoints)

		derivedService, err := r.getDerivedService(ctx, svc.Namespace, svc.Name, clusterId)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			// create derived Service if it doesn't exist
			if derivedService, err = r.createAndGetDerivedService(ctx, svcImport, clusterId, clusterImportedSvcPorts); err != nil {
				return err
			}
		}

		// update derived Service ports to match imported ports if necessary
		if err = r.updateDerivedService(ctx, derivedService, clusterImportedSvcPorts); err != nil {
			return err
		}

		// update EndpointSlices of this derived Service
		if err = r.updateEndpointSlices(ctx, svcImport, endpoints, derivedService, clusterId); err != nil {
			return err
		}

		derivedServices = append(derivedServices, derivedService)
	}

	// remove any existing derived services that do not have any endpoints in cloud map
	existingDerivedServices := &v1.ServiceList{}
	existingDerivedSvcErr := r.Client.List(ctx, existingDerivedServices, client.InNamespace(svcImport.Namespace), client.MatchingLabels{LabelDerivedServiceOriginatingName: svcImport.Name})
	if existingDerivedSvcErr != nil {
		r.Log.Error(existingDerivedSvcErr, "failed to list derived services")
		return existingDerivedSvcErr
	}
	for _, derivedService := range existingDerivedServices.Items {
		clusterId := derivedService.Labels[LabelSourceCluster]
		if _, ok := clusterIdToEndpointsMap[clusterId]; !ok {
			if err := r.DeleteDerivedServiceAndEndpointSlices(ctx, &derivedService); err != nil {
				return err
			}
		}
	}

	// update service import to match derived service clusterIPs and imported ports if necessary
	return r.updateServiceImport(ctx, svcImport, derivedServices, importedSvcPorts)
}

func (r *CloudMapReconciler) getServiceImport(ctx context.Context, namespace string, name string) (*multiclusterv1alpha1.ServiceImport, error) {
	existingServiceImport := &multiclusterv1alpha1.ServiceImport{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, existingServiceImport)
	return existingServiceImport, err
}

func (r *CloudMapReconciler) createAndGetServiceImport(ctx context.Context, svc *model.Service, servicePorts []*model.Port, clusterIds []string) (*multiclusterv1alpha1.ServiceImport, error) {
	toCreate := CreateServiceImportStruct(svc, clusterIds, servicePorts)
	if err := r.Client.Create(ctx, toCreate); err != nil {
		return nil, err
	}
	r.Log.Info("created ServiceImport", "namespace", svc.Namespace, "name", svc.Name)

	return r.getServiceImport(ctx, svc.Namespace, svc.Name)
}

func (r *CloudMapReconciler) getDerivedService(ctx context.Context, namespace string, name string, clusterId string) (*v1.Service, error) {
	derivedName := DerivedName(namespace, name, clusterId)
	existingService := &v1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: derivedName}, existingService)
	return existingService, err
}

func (r *CloudMapReconciler) createAndGetDerivedService(ctx context.Context, svcImport *multiclusterv1alpha1.ServiceImport, clusterId string, svcPorts []*model.Port) (*v1.Service, error) {
	toCreate := CreateDerivedServiceStruct(svcImport, svcPorts, clusterId)
	if err := r.Client.Create(ctx, toCreate); err != nil {
		return nil, err
	}
	r.Log.Info("created derived Service", "namespace", toCreate.Namespace, "name", toCreate.Name)

	return r.getDerivedService(ctx, svcImport.Namespace, svcImport.Name, clusterId)
}

func (r *CloudMapReconciler) updateEndpointSlices(ctx context.Context, svcImport *multiclusterv1alpha1.ServiceImport, desiredEndpoints []*model.Endpoint, svc *v1.Service, clusterId string) error {
	existingSlicesList := discovery.EndpointSliceList{}
	if err := r.Client.List(ctx, &existingSlicesList,
		client.InNamespace(svc.Namespace), client.MatchingLabels{discovery.LabelServiceName: svc.Name}); err != nil {
		return err
	}

	existingSlices := make([]*discovery.EndpointSlice, 0)
	for _, existingSlice := range existingSlicesList.Items {
		existingSlices = append(existingSlices, &existingSlice)
	}

	plan := EndpointSlicePlan{
		Current:           existingSlices,
		Desired:           desiredEndpoints,
		Service:           svc,
		ServiceImportName: svcImport.Name,
		ClusterId:         clusterId,
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

func (r *CloudMapReconciler) updateServiceImport(ctx context.Context, svcImport *multiclusterv1alpha1.ServiceImport, derivedServices []*v1.Service, importedSvcPorts []*model.Port) error {
	updateRequired := false

	clusterIPs := GetClusterIpsFromServices(derivedServices)
	if !IPsEqualIgnoreOrder(svcImport.Spec.IPs, clusterIPs) {
		r.Log.Info("ServiceImport IPs need update", "ServiceImport IPs", svcImport.Spec.IPs, "cluster IPs", clusterIPs)
		svcImport.Spec.IPs = clusterIPs
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
		serviceImportPorts := make([]multiclusterv1alpha1.ServicePort, 0)
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

func (r *CloudMapReconciler) DeleteDerivedServiceAndEndpointSlices(ctx context.Context, derivedService *v1.Service) error {
	// delete EndpointSlices
	if err := r.Client.DeleteAllOf(ctx, &discovery.EndpointSlice{}, client.InNamespace(derivedService.Namespace), client.MatchingLabels{discovery.LabelServiceName: derivedService.Name}); err != nil {
		return err
	}
	// delete Service
	r.Log.Info("deleting derived Service", "namespace", derivedService.Namespace, "name", derivedService.Name)
	return r.Client.Delete(ctx, derivedService)
}
