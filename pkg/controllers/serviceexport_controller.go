/*


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
	"context"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/api/v1alpha1"
)

const (
	K8sVersionAttr            = "K8S_CONTROLLER"
	ServiceExportFinalizer    = "multicluster.k8s.aws/service-export-finalizer"
	EndpointSliceServiceLabel = "kubernetes.io/service-name"
)

// ServiceExportReconciler reconciles a ServiceExport object
type ServiceExportReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	CloudMap cloudmap.ServiceDiscoveryClient
}

// +kubebuilder:rbac:groups="",resources=services,verbs=get
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=list;watch;create
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceexports,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceexports/finalizers,verbs=get;update

func (r *ServiceExportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r.Log.Info("reconciling ServiceExport", "Namespace", req.Namespace, "Name", req.NamespacedName)

	serviceExport := v1alpha1.ServiceExport{}
	if err := r.Client.Get(ctx, req.NamespacedName, &serviceExport); err != nil {
		r.Log.Error(err, "error fetching ServiceExport",
			"Namespace", serviceExport.Namespace, "Name", serviceExport.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Mark ServiceExport to be deleted, which is indicated by the deletion timestamp being set.
	isServiceExportMarkedForDelete := serviceExport.GetDeletionTimestamp() != nil

	service := v1.Service{}
	namespacedName := types.NamespacedName{Namespace: serviceExport.Namespace, Name: serviceExport.Name}
	if err := r.Client.Get(ctx, namespacedName, &service); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(err, "no Service found for ServiceExport",
				"Namespace", serviceExport.Namespace, "Name", serviceExport.Name)
			// Mark ServiceExport to be deleted, if the corresponding Service is not found
			isServiceExportMarkedForDelete = true
		} else {
			r.Log.Error(err, "error fetching service",
				"Namespace", serviceExport.Namespace, "Name", serviceExport.Name)
			return ctrl.Result{}, err
		}
	}

	// Check if the service export is marked to be deleted
	if isServiceExportMarkedForDelete {
		return r.handleDelete(ctx, &serviceExport)
	}

	return r.handleUpdate(ctx, &serviceExport, &service)
}

func (r *ServiceExportReconciler) handleUpdate(ctx context.Context, serviceExport *v1alpha1.ServiceExport, service *v1.Service) (ctrl.Result, error) {

	// Add the finalizer to the service export if not present, ensures the ServiceExport won't be deleted
	if !controllerutil.ContainsFinalizer(serviceExport, ServiceExportFinalizer) {
		controllerutil.AddFinalizer(serviceExport, ServiceExportFinalizer)
		if err := r.Update(ctx, serviceExport); err != nil {
			r.Log.Error(err, "error adding finalizer",
				"Namespace", serviceExport.Namespace, "Name", serviceExport.Name)
			return ctrl.Result{}, err
		}
	}

	r.Log.Info("updating Cloud Map service", "namespace", service.Namespace, "name", service.Name)
	cmService, err := r.createOrGetCloudMapService(ctx, service)
	if err != nil {
		r.Log.Error(err, "error fetching service from Cloud Map",
			"namespace", service.Namespace, "name", service.Name)
		return ctrl.Result{}, err
	}

	endpoints, err := r.extractEndpoints(ctx, service)
	if err != nil {
		r.Log.Error(err, "error extracting endpoints",
			"Namespace", serviceExport.Namespace, "Name", serviceExport.Name)
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
			r.Log.Error(err, "error registering endpoints to Cloud Map",
				"namespace", service.Namespace, "name", service.Name)
			return ctrl.Result{}, err
		}
	}

	if changes.HasDeletes() {
		if err := r.CloudMap.DeleteEndpoints(ctx, service.Namespace, service.Name, changes.Delete); err != nil {
			r.Log.Error(err, "error deleting endpoints from Cloud Map",
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
		if err := r.CloudMap.CreateService(ctx, service.Namespace, service.Name); err != nil {
			r.Log.Error(err, "error creating a new service in Cloud Map",
				"namespace", service.Namespace, "name", service.Name)
			return nil, err
		}
		if cmService, err = r.CloudMap.GetService(ctx, service.Namespace, service.Name); err != nil {
			return nil, err
		}
	}

	return cmService, nil
}

func (r *ServiceExportReconciler) handleDelete(ctx context.Context, serviceExport *v1alpha1.ServiceExport) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(serviceExport, ServiceExportFinalizer) {

		r.Log.Info("removing service export", "namespace", serviceExport.Namespace, "name", serviceExport.Name)

		cmService, err := r.CloudMap.GetService(ctx, serviceExport.Namespace, serviceExport.Name)
		if err != nil {
			r.Log.Error(err, "error fetching service from Cloud Map",
				"namespace", serviceExport.Namespace, "name", serviceExport.Name)
			return ctrl.Result{}, err
		}
		if cmService != nil {
			if err := r.CloudMap.DeleteEndpoints(ctx, cmService.Namespace, cmService.Name, cmService.Endpoints); err != nil {
				r.Log.Error(err, "error deleting endpoints from Cloud Map",
					"namespace", cmService.Namespace, "name", cmService.Name)
				return ctrl.Result{}, err
			}
		}

		// Remove finalizer. Once all finalizers have been
		// removed, the ServiceExport object will be deleted.
		controllerutil.RemoveFinalizer(serviceExport, ServiceExportFinalizer)
		if err := r.Update(ctx, serviceExport); err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *ServiceExportReconciler) extractEndpoints(ctx context.Context, svc *v1.Service) ([]*model.Endpoint, error) {
	result := make([]*model.Endpoint, 0)

	endpointSlices := discovery.EndpointSliceList{}
	err := r.Client.List(ctx, &endpointSlices,
		client.InNamespace(svc.Namespace), client.MatchingLabels{discovery.LabelServiceName: svc.Name})

	if err != nil {
		return nil, err
	}

	servicePortMap := make(map[string]model.Port)
	for _, svcPort := range svc.Spec.Ports {
		servicePortMap[svcPort.Name] = ServicePortToPort(svcPort)
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
						Id:           model.EndpointIdFromIPAddressAndPort(IP, port),
						IP:           IP,
						EndpointPort: port,
						ServicePort:  servicePortMap[*endpointPort.Name],
						Attributes:   attributes,
					})
				}
			}
		}
	}

	return result, nil
}

func (r *ServiceExportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceExport{}).
		// Watch for the changes to the EndpointSlice object. This object is bound to be
		// updated when Service or Deployment are updated. There is also a filtering logic
		// to enqueue those EndpointSlice event which have corresponding ServiceExport
		Watches(
			&source.Kind{Type: &discovery.EndpointSlice{}},
			handler.EnqueueRequestsFromMapFunc(r.endpointSliceEventHandler()),
			builder.WithPredicates(r.endpointSliceFilter()),
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
	svcExport := v1alpha1.ServiceExport{}
	if err := r.Client.Get(context.TODO(), ns, &svcExport); err != nil {
		return false
	}
	return true
}
