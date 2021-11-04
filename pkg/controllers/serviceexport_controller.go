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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/api/v1alpha1"
)

const (
	K8sVersionAttr         = "K8S_CONTROLLER"
	serviceExportFinalizer = "multicluster.k8s.aws/service-export-finalizer"
)

// ServiceExportReconciler reconciles a ServiceExport object
type ServiceExportReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Cloudmap cloudmap.ServiceDiscoveryClient
}

// +kubebuilder:rbac:groups="",resources=services,verbs=get
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=list;watch;create
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceexports,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceexports/finalizers,verbs=get;update

func (r *ServiceExportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log := r.Log.WithValues("serviceexport", req.NamespacedName)

	svcExport := v1alpha1.ServiceExport{}
	if err := r.Client.Get(ctx, req.NamespacedName, &svcExport); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if svcExport.DeletionTimestamp.IsZero() {
		return r.handleUpdate(ctx, log, &svcExport)
	} else {
		return r.handleDelete(ctx, log, &svcExport)
	}
}

func (r *ServiceExportReconciler) handleUpdate(ctx context.Context, log logr.Logger, svcExport *v1alpha1.ServiceExport) (ctrl.Result, error) {
	// add finalizer if not present
	if !controllerutil.ContainsFinalizer(svcExport, serviceExportFinalizer) {
		controllerutil.AddFinalizer(svcExport, serviceExportFinalizer)
		if err := r.Update(ctx, svcExport); err != nil {
			return ctrl.Result{}, err
		}
	}

	svc := v1.Service{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: svcExport.Namespace, Name: svcExport.Name}, &svc); err != nil {
		log.Error(err, "no service found for ServiceExport",
			"Namespace", svcExport.GetNamespace(), "Name", svcExport.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	endpoints, err := r.extractEndpoints(ctx, &svc)
	if err != nil {
		return ctrl.Result{}, err
	}

	changes := model.Changes{
		Create: endpoints,
	}

	log.Info("updating Cloud Map service", "namespace", svc.Namespace, "name", svc.Name)
	srv, err := r.Cloudmap.GetService(ctx, svc.Namespace, svc.Name)
	if err != nil {
		log.Error(err, "error when fetching service from Cloud Map API", "namespace", svc.Namespace, "name", svc.Name)
		return ctrl.Result{}, err
	}
	if srv == nil {
		if err := r.Cloudmap.CreateService(ctx, svc.Namespace, svc.Name); err != nil {
			log.Error(err, "error when creating new service in Cloud Map", "namespace", svc.Namespace, "name", svc.Name)
			return ctrl.Result{}, err
		}
	} else {
		// compute diff between Cloud Map and K8s and apply changes
		plan := model.Plan{
			Current: srv.Endpoints,
			Desired: endpoints,
		}
		changes = plan.CalculateChanges()
	}

	createRequired := len(changes.Create) > 0
	updateRequired := len(changes.Update) > 0
	deleteRequired := len(changes.Delete) > 0

	if createRequired || updateRequired {
		// merge creates and updates (Cloud Map RegisterEndpoints can handle both)
		upserts := append(changes.Create, changes.Update...)

		if err := r.Cloudmap.RegisterEndpoints(ctx, svc.Namespace, svc.Name, upserts); err != nil {
			log.Error(err, "error when registering endpoints to Cloud Map",
				"namespace", svc.Namespace, "name", svc.Name)
			return ctrl.Result{}, err
		}
	}

	if deleteRequired {
		if err := r.Cloudmap.DeleteEndpoints(ctx, svc.Namespace, svc.Name, changes.Delete); err != nil {
			log.Error(err, "error when deleting endpoints from Cloud Map",
				"namespace", srv.Namespace, "name", srv.Name)
			return ctrl.Result{}, err
		}
	}

	if !createRequired && !updateRequired && !deleteRequired {
		log.Info("no changes to export", "namespace", svc.Namespace, "name", svc.Name)
	}

	return ctrl.Result{}, nil
}

func (r *ServiceExportReconciler) handleDelete(ctx context.Context, log logr.Logger, svcExport *v1alpha1.ServiceExport) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(svcExport, serviceExportFinalizer) {

		log.Info("removing Cloud Map service", "namespace", svcExport.Namespace, "name", svcExport.Name)

		srv, err := r.Cloudmap.GetService(ctx, svcExport.Namespace, svcExport.Name)
		if err != nil {
			log.Error(err, "error when fetching service from Cloud Map API",
				"namespace", svcExport.Namespace, "name", svcExport.Name)
			return ctrl.Result{}, err
		}
		if srv != nil {
			if err := r.Cloudmap.DeleteEndpoints(ctx, srv.Namespace, srv.Name, srv.Endpoints); err != nil {
				log.Error(err, "error when deleting endpoints from Cloud Map",
					"namespace", srv.Namespace, "name", srv.Name)
				return ctrl.Result{}, err
			}
		}

		// remove finalizer
		controllerutil.RemoveFinalizer(svcExport, serviceExportFinalizer)
		if err := r.Update(ctx, svcExport); err != nil {
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

	for _, slice := range endpointSlices.Items {
		if slice.AddressType != discovery.AddressTypeIPv4 {
			return nil, fmt.Errorf("unsupported address type %s for service %s", slice.AddressType, svc.Name)
		}

		for _, port := range slice.Ports {
			for _, ep := range slice.Endpoints {
				for _, IP := range ep.Addresses {
					attributes := make(map[string]string, 0)
					if version.GetVersion() != "" {
						attributes[K8sVersionAttr] = version.PackageName + " " + version.GetVersion()
					}
					// TODO extract attributes - pod, node and other useful details if possible

					result = append(result, &model.Endpoint{
						Id:         model.EndpointIdFromIPAddress(IP),
						IP:         IP,
						Port:       *port.Port,
						Attributes: attributes,
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
		Complete(r)
}
