package controllers

import (
	"context"
	"fmt"

	aboutv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/about/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/version"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	multiclusterv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
)

const (
	K8sVersionAttr            = "K8S_CONTROLLER"
	ServiceExportFinalizer    = "multicluster.k8s.aws/service-export-finalizer"
	EndpointSliceServiceLabel = "kubernetes.io/service-name"
)

// ServiceExportReconciler reconciles a ServiceExport object
type ServiceExportReconciler struct {
	Client       client.Client
	Log          common.Logger
	Scheme       *runtime.Scheme
	CloudMap     cloudmap.ServiceDiscoveryClient
	ClusterUtils common.ClusterUtils
}

// +kubebuilder:rbac:groups="",resources=services,verbs=get
// +kubebuilder:rbac:groups=about.k8s.io,resources=clusterproperties,verbs=create;get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=list;watch;create
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceexports,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceexports/finalizers,verbs=get;update

func (r *ServiceExportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	namespace := req.Namespace
	name := req.NamespacedName
	r.Log.Debug("reconciling ServiceExport", "Namespace", namespace, "Name", name)

	var err error
	clusterId, err := r.ClusterUtils.GetClusterId(ctx)
	if err != nil {
		r.Log.Error(err, "unable to retrieve clusterId")
		return ctrl.Result{}, err
	}
	clusterSetId, err := r.ClusterUtils.GetClusterSetId(ctx)
	if err != nil {
		r.Log.Error(err, "unable to retrieve clusterSetId")
		return ctrl.Result{}, err
	}
	r.Log.Debug("ClusterId and ClusterSetId found", "ClusterId", clusterId, "ClusterSetId", clusterSetId)

	serviceExport := multiclusterv1alpha1.ServiceExport{}
	if err := r.Client.Get(ctx, name, &serviceExport); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Debug("no ServiceExport found",
				"Namespace", namespace, "Name", name)
		} else {
			r.Log.Error(err, "error fetching ServiceExport",
				"Namespace", namespace, "Name", name)
		}
		return ctrl.Result{}, nil
	}

	// Mark ServiceExport to be deleted, which is indicated by the deletion timestamp being set.
	isServiceExportMarkedForDelete := serviceExport.GetDeletionTimestamp() != nil

	service := v1.Service{}
	namespacedName := types.NamespacedName{Namespace: serviceExport.Namespace, Name: serviceExport.Name}
	if err := r.Client.Get(ctx, namespacedName, &service); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("no Service found, deleting the ServiceExport",
				"Namespace", serviceExport.Namespace, "Name", serviceExport.Name)
			// Mark ServiceExport to be deleted, if the corresponding Service is not found
			isServiceExportMarkedForDelete = true
		} else {
			r.Log.Error(err, "error fetching Service",
				"Namespace", serviceExport.Namespace, "Name", serviceExport.Name)
			return ctrl.Result{}, nil
		}
	}

	// Check if the service export is marked to be deleted
	if isServiceExportMarkedForDelete {
		return r.handleDelete(ctx, &serviceExport)
	}

	return r.handleUpdate(ctx, &serviceExport, &service)
}

func (r *ServiceExportReconciler) handleUpdate(ctx context.Context, serviceExport *multiclusterv1alpha1.ServiceExport, service *v1.Service) (ctrl.Result, error) {
	// Add the finalizer to the service export if not present, ensures the ServiceExport won't be deleted
	if !controllerutil.ContainsFinalizer(serviceExport, ServiceExportFinalizer) {
		controllerutil.AddFinalizer(serviceExport, ServiceExportFinalizer)
		if err := r.Client.Update(ctx, serviceExport); err != nil {
			r.Log.Error(err, "error adding finalizer",
				"Namespace", serviceExport.Namespace, "Name", serviceExport.Name)
			return ctrl.Result{}, err
		}
	}

	if len(serviceExport.GetOwnerReferences()) == 0 {
		err := controllerutil.SetControllerReference(service, serviceExport, r.Scheme)
		if err == nil {
			err = r.Client.Update(ctx, serviceExport)
		}
		if err != nil {
			r.Log.Error(err, "error setting Service as an owner of the ServiceExport",
				"namespace", service.Namespace, "name", service.Name)
			return ctrl.Result{}, err
		}
	}

	r.Log.Info("updating Cloud Map service", "namespace", service.Namespace, "name", service.Name)
	cmService, err := r.createOrGetCloudMapService(ctx, service)
	if err != nil {
		r.Log.Error(err, "error fetching Service from Cloud Map",
			"namespace", service.Namespace, "name", service.Name)
		return ctrl.Result{}, err
	}

	endpoints, err := r.extractEndpoints(ctx, service, serviceExport)
	if err != nil {
		r.Log.Error(err, "error extracting Endpoints",
			"namespace", serviceExport.Namespace, "name", serviceExport.Name)
		return ctrl.Result{}, err
	}

	// Compute diff between Cloud Map and K8s endpoints, and apply changes
	plan := model.Plan{
		Current: cmService.Endpoints,
		Desired: endpoints,
	}
	changes := plan.CalculateChanges()

	if changes.HasUpdates() {
		// merge creates and updates (Cloud Map RegisterEndpoints can handle both)
		upserts := changes.Create
		upserts = append(upserts, changes.Update...)

		if err := r.CloudMap.RegisterEndpoints(ctx, service.Namespace, service.Name, upserts); err != nil {
			r.Log.Error(err, "error registering Endpoints to Cloud Map",
				"namespace", service.Namespace, "name", service.Name)
			return ctrl.Result{}, err
		}
	}

	if changes.HasDeletes() {
		if err := r.CloudMap.DeleteEndpoints(ctx, service.Namespace, service.Name, changes.Delete); err != nil {
			r.Log.Error(err, "error deleting Endpoints from Cloud Map",
				"namespace", cmService.Namespace, "name", cmService.Name)
			return ctrl.Result{}, err
		}
	}

	if changes.IsNone() {
		r.Log.Info("no changes to export to Cloud Map", "namespace", service.Namespace, "name", service.Name)
	}

	return ctrl.Result{}, nil
}

func (r *ServiceExportReconciler) createOrGetCloudMapService(ctx context.Context, service *v1.Service) (*model.Service, error) {
	cmService, err := r.CloudMap.GetService(ctx, service.Namespace, service.Name)
	if err != nil {
		return nil, err
	}

	if cmService == nil {
		err = r.CloudMap.CreateService(ctx, service.Namespace, service.Name)
		if err != nil {
			r.Log.Error(err, "error creating a new Service in Cloud Map",
				"namespace", service.Namespace, "name", service.Name)
			return nil, err
		}
		if cmService, err = r.CloudMap.GetService(ctx, service.Namespace, service.Name); err != nil {
			return nil, err
		}
	}

	return cmService, nil
}

func (r *ServiceExportReconciler) handleDelete(ctx context.Context, serviceExport *multiclusterv1alpha1.ServiceExport) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(serviceExport, ServiceExportFinalizer) {
		r.Log.Info("removing service export", "namespace", serviceExport.Namespace, "name", serviceExport.Name)

		cmService, err := r.CloudMap.GetService(ctx, serviceExport.Namespace, serviceExport.Name)
		if err != nil {
			r.Log.Error(err, "error fetching Service from Cloud Map",
				"namespace", serviceExport.Namespace, "name", serviceExport.Name)
			return ctrl.Result{}, err
		}
		if cmService != nil {
			if err := r.CloudMap.DeleteEndpoints(ctx, cmService.Namespace, cmService.Name, cmService.Endpoints); err != nil {
				r.Log.Error(err, "error deleting Endpoints from Cloud Map",
					"namespace", cmService.Namespace, "name", cmService.Name)
				return ctrl.Result{}, err
			}
		}

		// Remove finalizer. Once all finalizers have been
		// removed, the ServiceExport object will be deleted.
		controllerutil.RemoveFinalizer(serviceExport, ServiceExportFinalizer)
		if err := r.Client.Update(ctx, serviceExport); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceExportReconciler) extractEndpoints(ctx context.Context, svc *v1.Service, svcExport *multiclusterv1alpha1.ServiceExport) ([]*model.Endpoint, error) {
	result := make([]*model.Endpoint, 0)

	endpointSlices := discovery.EndpointSliceList{}
	err := r.Client.List(ctx, &endpointSlices,
		client.InNamespace(svc.Namespace), client.MatchingLabels{discovery.LabelServiceName: svc.Name})

	if err != nil {
		return nil, err
	}

	serviceType := ExtractServiceType(svc)

	servicePortMap := make(map[string]model.Port)
	for _, svcPort := range svc.Spec.Ports {
		servicePortMap[svcPort.Name] = ServicePortToPort(svcPort)
	}

	clusterId, err := r.ClusterUtils.GetClusterId(ctx)
	if err != nil {
		r.Log.Error(err, "unable to retrieve clusterId")
		return nil, err
	}
	clusterSetId, err := r.ClusterUtils.GetClusterSetId(ctx)
	if err != nil {
		r.Log.Error(err, "unable to retrieve clusterSetId")
		return nil, err
	}

	var svcExportCreationTimestamp int64 = 0
	if !svcExport.ObjectMeta.CreationTimestamp.IsZero() {
		svcExportCreationTimestamp = svcExport.ObjectMeta.CreationTimestamp.Time.UnixMilli()
	}

	for _, slice := range endpointSlices.Items {
		if slice.AddressType != discovery.AddressTypeIPv4 {
			return nil, fmt.Errorf("unsupported address type %s for service %s", slice.AddressType, svc.Name)
		}
		for _, endpointPort := range slice.Ports {
			for _, endpoint := range slice.Endpoints {
				for _, IP := range endpoint.Addresses {
					attributes := make(map[string]string)
					if version.GetVersion() != "" {
						attributes[K8sVersionAttr] = version.PackageName + " " + version.GetVersion()
					}

					// TODO extract attributes - pod, node and other useful details if possible

					port := EndpointPortToPort(endpointPort)
					result = append(result, &model.Endpoint{
						Id:                         model.EndpointIdFromIPAddressAndPort(IP, port),
						IP:                         IP,
						EndpointPort:               port,
						ServicePort:                servicePortMap[*endpointPort.Name],
						ClusterId:                  clusterId,
						ClusterSetId:               clusterSetId,
						ServiceType:                serviceType,
						SvcExportCreationTimestamp: svcExportCreationTimestamp,
						Attributes:                 attributes,
					})
				}
			}
		}
	}

	return result, nil
}

func (r *ServiceExportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiclusterv1alpha1.ServiceExport{}).
		// Watch for the changes to the EndpointSlice object. This object is bound to be
		// updated when Service or Deployment are updated. There is also a filtering logic
		// to enqueue those EndpointSlice event which have corresponding ServiceExport
		Watches(
			&source.Kind{Type: &discovery.EndpointSlice{}},
			handler.EnqueueRequestsFromMapFunc(r.endpointSliceEventHandler()),
			builder.WithPredicates(r.endpointSliceFilter()),
		).
		// Watch for changes to ClusterProperty objects. If a ClusterProperty object is
		// created, updated or deleted, the controller will reconcile all service exports
		Watches(
			&source.Kind{Type: &aboutv1alpha1.ClusterProperty{}},
			handler.EnqueueRequestsFromMapFunc(r.clusterPropertyEventHandler()),
		).
		Complete(r)
}

func (r *ServiceExportReconciler) endpointSliceEventHandler() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		labels := object.GetLabels()
		serviceName := labels[EndpointSliceServiceLabel]
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Name:      serviceName,
				Namespace: object.GetNamespace(),
			}},
		}
	}
}

func (r *ServiceExportReconciler) clusterPropertyEventHandler() handler.MapFunc {
	// Return reconcile requests for all service exports
	return func(object client.Object) []reconcile.Request {
		serviceExports := &multiclusterv1alpha1.ServiceExportList{}
		if err := r.Client.List(context.TODO(), serviceExports); err != nil {
			r.Log.Error(err, "error listing services")
			return nil
		}

		result := make([]reconcile.Request, 0)
		for _, serviceExport := range serviceExports.Items {
			result = append(result, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      serviceExport.Name,
				Namespace: serviceExport.Namespace,
			}})
		}
		return result
	}
}

func (r *ServiceExportReconciler) endpointSliceFilter() predicate.Funcs {
	return predicate.Funcs{
		GenericFunc: func(e event.GenericEvent) bool {
			return r.doesEndpointSliceHaveServiceExport(e.Object)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return r.doesEndpointSliceHaveServiceExport(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.doesEndpointSliceHaveServiceExport(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.doesEndpointSliceHaveServiceExport(e.Object)
		},
	}
}

func (r *ServiceExportReconciler) doesEndpointSliceHaveServiceExport(object client.Object) bool {
	labels := object.GetLabels()
	serviceName := labels[EndpointSliceServiceLabel]
	ns := types.NamespacedName{
		Name:      serviceName,
		Namespace: object.GetNamespace(),
	}
	svcExport := multiclusterv1alpha1.ServiceExport{}
	if err := r.Client.Get(context.TODO(), ns, &svcExport); err != nil {
		return false
	}
	return true
}
