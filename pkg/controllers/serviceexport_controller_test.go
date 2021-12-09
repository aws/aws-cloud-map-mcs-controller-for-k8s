package controllers

import (
	"context"

	cloudmapMock "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/api/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	testing2 "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"

	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceExportReconciler_Reconcile_NewServiceExport(t *testing.T) {
	// create a fake controller client and add some objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(getServiceExportScheme()).
		WithObjects(k8sServiceForTest(), serviceExportForTest()).
		WithLists(&discovery.EndpointSliceList{
			Items: []discovery.EndpointSlice{*endpointSliceForTest()},
		}).
		Build()

	// create a mock cloudmap service discovery client
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mock := cloudmapMock.NewMockServiceDiscoveryClient(mockController)
	// expected interactions with the Cloud Map client
	// The first get call is expected to return nil, then second call after the creation of service is
	// supposed to return the value
	first := mock.EXPECT().GetService(gomock.Any(), test.HttpNsName, test.SvcName).Return(nil, nil)
	second := mock.EXPECT().GetService(gomock.Any(), test.HttpNsName, test.SvcName).
		Return(&model.Service{Namespace: test.HttpNsName, Name: test.SvcName}, nil)
	gomock.InOrder(first, second)
	mock.EXPECT().CreateService(gomock.Any(), test.HttpNsName, test.SvcName).Return(nil).Times(1)
	mock.EXPECT().RegisterEndpoints(gomock.Any(), test.HttpNsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1()}).Return(nil).Times(1)

	reconciler := getServiceExportReconciler(t, mock, fakeClient)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: test.HttpNsName,
			Name:      test.SvcName,
		},
	}

	got, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Errorf("Reconcile() error = %v", err)
		return
	}
	assert.Equal(t, ctrl.Result{}, got, "Result should be empty")

	serviceExport := &v1alpha1.ServiceExport{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.HttpNsName, Name: test.SvcName}, serviceExport)
	assert.NoError(t, err)
	assert.Contains(t, serviceExport.Finalizers, ServiceExportFinalizer, "Finalizer added to the service export")
}

func TestServiceExportReconciler_Reconcile_ExistingServiceExport(t *testing.T) {
	// create a fake controller client and add some objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(getServiceExportScheme()).
		WithObjects(k8sServiceForTest(), serviceExportForTest()).
		WithLists(&discovery.EndpointSliceList{
			Items: []discovery.EndpointSlice{*endpointSliceForTest()},
		}).
		Build()

	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mock := cloudmapMock.NewMockServiceDiscoveryClient(mockController)

	// GetService from Cloudmap returns endpoint1 and endpoint2
	mock.EXPECT().GetService(gomock.Any(), test.HttpNsName, test.SvcName).
		Return(test.GetTestService(), nil)
	// call to delete the endpoint not present in the k8s cluster
	mock.EXPECT().DeleteEndpoints(gomock.Any(), test.HttpNsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint2()}).Return(nil).Times(1)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: test.HttpNsName,
			Name:      test.SvcName,
		},
	}

	reconciler := getServiceExportReconciler(t, mock, fakeClient)

	got, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Errorf("Reconcile() error = %v", err)
		return
	}
	assert.Equal(t, ctrl.Result{}, got, "Result should be empty")

	serviceExport := &v1alpha1.ServiceExport{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.HttpNsName, Name: test.SvcName}, serviceExport)
	assert.NoError(t, err)
	assert.Contains(t, serviceExport.Finalizers, ServiceExportFinalizer, "Finalizer added to the service export")
}

func TestServiceExportReconciler_Reconcile_DeleteExistingService(t *testing.T) {
	// create a fake controller client and add some objects
	serviceExportObj := serviceExportForTest()
	// Add finalizer string to the service
	serviceExportObj.Finalizers = []string{ServiceExportFinalizer}
	fakeClient := fake.NewClientBuilder().
		WithScheme(getServiceExportScheme()).
		WithObjects(serviceExportObj).
		WithLists(&discovery.EndpointSliceList{
			Items: []discovery.EndpointSlice{*endpointSliceForTest()},
		}).
		Build()

	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mock := cloudmapMock.NewMockServiceDiscoveryClient(mockController)

	// GetService from Cloudmap returns endpoint1 and endpoint2
	mock.EXPECT().GetService(gomock.Any(), test.HttpNsName, test.SvcName).
		Return(test.GetTestService(), nil)
	// call to delete the endpoint in the cloudmap
	mock.EXPECT().DeleteEndpoints(gomock.Any(), test.HttpNsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()}).Return(nil).Times(1)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: test.HttpNsName,
			Name:      test.SvcName,
		},
	}

	reconciler := getServiceExportReconciler(t, mock, fakeClient)

	got, err := reconciler.Reconcile(context.Background(), request)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, got, "Result should be empty")

	serviceExport := &v1alpha1.ServiceExport{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.HttpNsName, Name: test.SvcName}, serviceExport)
	assert.NoError(t, err)
	assert.Empty(t, serviceExport.Finalizers, "Finalizer removed from the service export")
}

func getServiceExportScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.ServiceExport{})
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1.Service{})
	scheme.AddKnownTypes(discovery.SchemeGroupVersion, &discovery.EndpointSlice{}, &discovery.EndpointSliceList{})
	return scheme
}

func getServiceExportReconciler(t *testing.T, mockClient *cloudmapMock.MockServiceDiscoveryClient, client client.Client) *ServiceExportReconciler {
	return &ServiceExportReconciler{
		Client:   client,
		Log:      common.NewLoggerWithLogr(testing2.TestLogger{T: t}),
		Scheme:   client.Scheme(),
		CloudMap: mockClient,
	}
}
