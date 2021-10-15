package controllers

import (
	"context"
	cloudmapmock "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/api/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	testing2 "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestServiceExportReconciler_Reconcile_NewServiceExport(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	expectedService := model.Service{
		Namespace: "my-namespace",
		Name:      "exported-service",
		Endpoints: []*model.Endpoint{{
			Id:         "1_1_1_1",
			IP:         "1.1.1.1",
			Port:       80,
			Attributes: map[string]string{},
		}},
	}

	cloudmapMock := cloudmapmock.NewMockServiceDiscoveryClient(mockController)
	// expected interactions with the Cloud Map client
	cloudmapMock.EXPECT().GetService(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	cloudmapMock.EXPECT().CreateService(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	cloudmapMock.EXPECT().RegisterEndpoints(gomock.Any(), gomock.Eq(&expectedService)).Return(nil).Times(1)

	reconciler := setupServiceExportReconciler(t, cloudmapMock)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "my-namespace",
			Name:      "exported-service",
		},
	}

	got, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Errorf("Reconcile() error = %v", err)
		return
	}
	assert.Equal(t, ctrl.Result{}, got, "Result should be empty")
}

func TestServiceExportReconciler_Reconcile_ExistingServiceNewEndpoint(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	emptyService := model.Service{
		Namespace: "my-namespace",
		Name:      "exported-service",
	}

	expectedService := model.Service{
		Namespace: "my-namespace",
		Name:      "exported-service",
		Endpoints: []*model.Endpoint{{
			Id:         "1_1_1_1",
			IP:         "1.1.1.1",
			Port:       80,
			Attributes: map[string]string{},
		}},
	}

	cloudmapMock := cloudmapmock.NewMockServiceDiscoveryClient(mockController)

	// expected interactions with the Cloud Map client
	cloudmapMock.EXPECT().GetService(gomock.Any(), gomock.Any(), gomock.Any()).Return(&emptyService, nil)
	cloudmapMock.EXPECT().RegisterEndpoints(gomock.Any(), gomock.Eq(&expectedService)).Return(nil).Times(1)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "my-namespace",
			Name:      "exported-service",
		},
	}

	reconciler := setupServiceExportReconciler(t, cloudmapMock)

	got, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Errorf("Reconcile() error = %v", err)
		return
	}
	assert.Equal(t, ctrl.Result{}, got, "Result should be empty")
}

func setupServiceExportReconciler(t *testing.T, cloudmapMock cloudmap.ServiceDiscoveryClient) *ServiceExportReconciler {
	k8sClient := setupK8sClient()

	return &ServiceExportReconciler{
		Client:   k8sClient,
		Log:      testing2.TestLogger{T: t},
		Scheme:   k8sClient.Scheme(),
		Cloudmap: cloudmapMock,
	}
}

func setupK8sClient() client.Client {
	// ServiceExport object
	serviceExport := &v1alpha1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exported-service",
			Namespace: "my-namespace",
		},
	}

	// Service object
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exported-service",
			Namespace: "my-namespace",
		},
	}

	// EndpointSlice object
	port := int32(80)
	endpointSlice := &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "slice-id",
			Labels:    map[string]string{discovery.LabelServiceName: "exported-service"},
		},
		AddressType: discovery.AddressTypeIPv4,
		Endpoints: []discovery.Endpoint{{
			Addresses: []string{"1.1.1.1"},
		}},
		Ports: []discovery.EndpointPort{{Port: &port}},
	}
	endpointSliceList := &discovery.EndpointSliceList{
		Items: []discovery.EndpointSlice{*endpointSlice},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.GroupVersion, serviceExport)
	scheme.AddKnownTypes(v1.SchemeGroupVersion, service)
	scheme.AddKnownTypes(discovery.SchemeGroupVersion, endpointSlice)
	scheme.AddKnownTypes(discovery.SchemeGroupVersion, endpointSliceList)

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(serviceExport, service).
		WithLists(endpointSliceList).
		Build()
}
