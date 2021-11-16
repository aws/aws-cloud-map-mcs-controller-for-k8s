package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/api/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	// TODO move to configuration
	syncPeriod = 2 * time.Second

	// DerivedServiceAnnotation annotates a ServiceImport with derived Service name
	DerivedServiceAnnotation = "multicluster.k8s.aws/derived-service"

	// LabelServiceImportName indicates the name of the multi-cluster service that an EndpointSlice belongs to.
	LabelServiceImportName = "multicluster.kubernetes.io/service-name"
)

// CloudMapReconciler reconciles state of Cloud Map services with local ServiceImport objects
type CloudMapReconciler struct {
	client.Client
	Cloudmap cloudmap.ServiceDiscoveryClient
	logr.Logger
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=create;get;list;watch
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=list;get;create;watch
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceimports,verbs=create;get;list;watch;update;patch;delete

// Start implements manager.Runnable
func (r *CloudMapReconciler) Start(ctx context.Context) error {
	ticker := time.NewTicker(syncPeriod)
	defer ticker.Stop()
	for {
		if err := r.Reconcile(ctx); err != nil {
			// just log the error and continue running
			r.Logger.Error(err, "Cloud Map reconciliation error")
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			r.Logger.Info("terminating CloudMapReconciler")
			return nil
		}
	}
}

// Reconcile triggers a single reconciliation round
func (r *CloudMapReconciler) Reconcile(ctx context.Context) error {

	namespaces := v1.NamespaceList{}
	if err := r.Client.List(ctx, &namespaces); err != nil {
		r.Logger.Error(err, "unable to list namespaces")
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
	r.Logger.Info("syncing namespace", "namespace", namespaceName)

	desiredServices, err := r.Cloudmap.ListServices(ctx, namespaceName)
	if err != nil {
		return err
	}

	serviceImports := v1alpha1.ServiceImportList{}
	if err := r.Client.List(ctx, &serviceImports, client.InNamespace(namespaceName)); err != nil {
		r.Logger.Error(err, "failed to reconcile namespace", "namespace", namespaceName)
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

		if err := r.reconcileService(ctx, svc); err != nil {
			r.Logger.Error(err, "error when syncing service", "namespace", svc.Namespace, "name", svc.Name)
		}
		delete(existingImportsMap, svc.Namespace+"/"+svc.Name)
	}

	// delete remaining imports that have not been matched
	for _, i := range existingImportsMap {
		if err := r.Client.Delete(ctx, &i); err != nil {
			r.Logger.Error(err, "error deleting ServiceImport", "namespace", i.Namespace, "name", i.Name)
			continue
		}
		r.Logger.Info("delete ServiceImport", "namespace", i.Namespace, "name", i.Name)
	}

	return nil
}

func (r *CloudMapReconciler) reconcileService(ctx context.Context, svc *model.Service) error {
	r.Logger.Info("syncing service", "namespace", svc.Namespace, "service", svc.Name)

	svcImport, err := r.getServiceImport(ctx, svc.Namespace, svc.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// create ServiceImport if it doesn't exist
		if svcImport, err = r.createAndGetServiceImport(ctx, svc.Namespace, svc.Name); err != nil {
			return err
		}
	}

	derivedService, err := r.getDerivedService(ctx, svc.Namespace, svcImport.Annotations[DerivedServiceAnnotation])
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// create derived Service if it doesn't exist
		if derivedService, err = r.createAndGetDerivedService(ctx, svc, svcImport); err != nil {
			return err
		}
	}

	// update ServiceImport to match IP and port of previously created service
	if err = r.updateServiceImport(ctx, svcImport, derivedService); err != nil {
		return err
	}

	err = r.updateEndpointSlices(ctx, svcImport, svc, derivedService)
	if err != nil {
		return err
	}

	return nil
}

func (r *CloudMapReconciler) getServiceImport(ctx context.Context, namespace string, name string) (*v1alpha1.ServiceImport, error) {
	existingServiceImport := &v1alpha1.ServiceImport{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, existingServiceImport)
	return existingServiceImport, err
}

func (r *CloudMapReconciler) createAndGetServiceImport(ctx context.Context, namespace string, name string) (*v1alpha1.ServiceImport, error) {
	imp := &v1alpha1.ServiceImport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name,
			Annotations: map[string]string{DerivedServiceAnnotation: DerivedName(namespace, name)},
		},
		Spec: v1alpha1.ServiceImportSpec{
			IPs:   []string{},
			Type:  v1alpha1.ClusterSetIP,
			Ports: []v1alpha1.ServicePort{},
		},
	}

	if err := r.Client.Create(ctx, imp); err != nil {
		return nil, err
	}
	r.Logger.Info("created ServiceImport", "namespace", imp.Namespace, "name", imp.Name)

	return r.getServiceImport(ctx, namespace, name)
}

func (r *CloudMapReconciler) getDerivedService(ctx context.Context, namespace string, name string) (*v1.Service, error) {
	existingService := &v1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, existingService)
	return existingService, err
}

func (r *CloudMapReconciler) createAndGetDerivedService(ctx context.Context, svc *model.Service, svcImport *v1alpha1.ServiceImport) (*v1.Service, error) {
	toCreate := createDerivedServiceStruct(svc, svcImport)
	if err := r.Client.Create(ctx, toCreate); err != nil {
		return nil, err
	}
	r.Logger.Info("created derived Service", "namespace", toCreate.Namespace, "name", toCreate.Name)

	return r.getDerivedService(ctx, svc.Namespace, svcImport.Annotations[DerivedServiceAnnotation])
}

func (r *CloudMapReconciler) updateEndpointSlices(ctx context.Context, svcImport *v1alpha1.ServiceImport, cloudMapService *model.Service, svc *v1.Service) error {
	existingSlicesList := discovery.EndpointSliceList{}
	var existingSlices []*discovery.EndpointSlice
	if err := r.Client.List(ctx, &existingSlicesList,
		client.InNamespace(svc.Namespace), client.MatchingLabels{discovery.LabelServiceName: svc.Name}); err != nil {
		return err
	}
	if len(existingSlicesList.Items) == 0 {
		// create new endpoint slice
		existingSlices = createEndpointSlicesStruct(svcImport, cloudMapService, svc)
		for _, slice := range existingSlices {
			if err := r.Client.Create(ctx, slice); err != nil {
				return err
			}
			r.Logger.Info("created EndpointSlice", "namespace", slice.Namespace, "name", slice.Name)
		}
	}

	// TODO check existing slices match Cloud Map endpoints

	return nil
}

// DerivedName computes the "placeholder" name for the imported service
func DerivedName(namespace string, name string) string {
	hash := sha256.New()
	hash.Write([]byte(namespace + name))
	return "imported-" + strings.ToLower(base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(hash.Sum(nil)))[:10]
}

func createDerivedServiceStruct(svc *model.Service, svcImport *v1alpha1.ServiceImport) *v1.Service {
	ownerRef := metav1.NewControllerRef(svcImport, schema.GroupVersionKind{
		Version: svcImport.TypeMeta.APIVersion,
		Kind:    svcImport.TypeMeta.Kind,
	})

	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       svcImport.Namespace,
			Name:            svcImport.Annotations[DerivedServiceAnnotation],
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
		Spec: v1.ServiceSpec{
			Type:  v1.ServiceTypeClusterIP,
			Ports: extractServicePorts(svc),
		},
	}
}

func createEndpointSlicesStruct(svcImport *v1alpha1.ServiceImport, cloudMapSvc *model.Service, svc *v1.Service) []*discovery.EndpointSlice {
	slices := make([]*discovery.EndpointSlice, 0)

	t := true

	endpoints := make([]discovery.Endpoint, 0)
	for _, ep := range cloudMapSvc.Endpoints {
		endpoints = append(endpoints, discovery.Endpoint{
			Addresses: []string{ep.IP},
			Conditions: discovery.EndpointConditions{
				Ready: &t,
			},
			TargetRef: &v1.ObjectReference{
				Kind:            "Service",
				Namespace:       svc.Namespace,
				Name:            svc.Name,
				UID:             svc.ObjectMeta.UID,
				ResourceVersion: svc.ObjectMeta.ResourceVersion,
			},
		})
	}

	// TODO split slices in case there are more than 1000 endpoints
	// see https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/0752-endpointslices/README.md

	slices = append(slices, &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				discovery.LabelServiceName: svc.Name,       // derived Service name
				LabelServiceImportName:     svcImport.Name, // original ServiceImport name
			},
			GenerateName: svc.Name + "-",
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(svc, schema.GroupVersionKind{
				Version: svc.TypeMeta.APIVersion,
				Kind:    svc.TypeMeta.Kind,
			})},
			Namespace: svc.Namespace,
		},
		AddressType: discovery.AddressTypeIPv4,
		Endpoints:   endpoints,
		Ports:       extractEndpointPorts(cloudMapSvc),
	})

	return slices
}

func extractServicePorts(svc *model.Service) []v1.ServicePort {
	uniquePorts := make(map[string]model.Port)
	for _, ep := range svc.Endpoints {
		uniquePorts[ep.ServicePort.GetID()] = ep.ServicePort
	}

	servicePorts := make([]v1.ServicePort, 0, len(uniquePorts))
	for _, servicePort := range uniquePorts {
		servicePorts = append(servicePorts, PortToServicePort(servicePort))
	}

	return servicePorts
}

func extractEndpointPorts(svc *model.Service) []discovery.EndpointPort {
	uniquePorts := make(map[string]model.Port)
	for _, ep := range svc.Endpoints {
		uniquePorts[ep.EndpointPort.GetID()] = ep.EndpointPort
	}

	endpointPorts := make([]discovery.EndpointPort, 0, len(uniquePorts))
	for _, endpointPort := range uniquePorts {
		endpointPorts = append(endpointPorts, PortToEndpointPort(endpointPort))
	}
	return endpointPorts
}

func (r *CloudMapReconciler) updateServiceImport(ctx context.Context, svcImport *v1alpha1.ServiceImport, svc *v1.Service) error {
	if len(svcImport.Spec.IPs) != 1 || svcImport.Spec.IPs[0] != svc.Spec.ClusterIP || !portsEqual(svcImport, svc) {
		svcImport.Spec.IPs = []string{svc.Spec.ClusterIP}

		svcImport.Spec.Ports = make([]v1alpha1.ServicePort, 0)
		for _, p := range svc.Spec.Ports {
			svcImport.Spec.Ports = append(svcImport.Spec.Ports, servicePortToServiceImport(p))
		}
		if err := r.Update(ctx, svcImport); err != nil {
			return err
		}
		r.Logger.Info("updated ServiceImport",
			"namespace", svcImport.Namespace, "name", svcImport.Name,
			"IP", svcImport.Spec.IPs, "ports", svcImport.Spec.Ports)
	}

	return nil
}

func portsEqual(svcImport *v1alpha1.ServiceImport, svc *v1.Service) bool {
	impPorts := svcImport.Spec.Ports
	svcPorts := make([]v1alpha1.ServicePort, 0)
	for _, p := range svc.Spec.Ports {
		svcPorts = append(svcPorts, servicePortToServiceImport(p))
	}

	return reflect.DeepEqual(impPorts, svcPorts)
}

func servicePortToServiceImport(port v1.ServicePort) v1alpha1.ServicePort {
	return v1alpha1.ServicePort{
		Name:        port.Name,
		Protocol:    port.Protocol,
		AppProtocol: port.AppProtocol,
		Port:        port.Port,
	}
}
