package controllers

import (
	"context"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/api/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	testing2 "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestServiceExportReconciler_Reconcile_NewServiceExport(t *testing.T) {
	// create a fake controller client and add some objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(getServiceExportScheme()).
		WithObjects(testServiceObj(), testServiceExportObj()).
		WithLists(testEndpointSliceObj()).
		Build()

	// create a mock cloudmap service discovery client
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mock := cloudmap.NewMockServiceDiscoveryClient(mockController)
	// expected interactions with the Cloud Map client
	// The first get call is expected to return nil, then second call after the creation of service is
	// supposed to return the value
	first := mock.EXPECT().GetService(gomock.Any(), test.NsName, test.SvcName).Return(nil, nil)
	second := mock.EXPECT().GetService(gomock.Any(), test.NsName, test.SvcName).
		Return(&model.Service{Namespace: test.NsName, Name: test.SvcName}, nil)
	gomock.InOrder(first, second)
	mock.EXPECT().CreateService(gomock.Any(), test.NsName, test.SvcName).Return(nil).Times(1)
	mock.EXPECT().RegisterEndpoints(gomock.Any(), test.NsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1()}).Return(nil).Times(1)

	reconciler := getServiceExportReconciler(t, mock, fakeClient)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: test.NsName,
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
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.NsName, Name: test.SvcName}, serviceExport)
	assert.NoError(t, err)
	assert.Contains(t, serviceExport.Finalizers, ServiceExportFinalizer, "Finalizer added to the service export")
}

func TestServiceExportReconciler_Reconcile_ExistingServiceExport(t *testing.T) {
	// create a fake controller client and add some objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(getServiceExportScheme()).
		WithObjects(testServiceObj(), testServiceExportObj()).
		WithLists(testEndpointSliceObj()).
		Build()

	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mock := cloudmap.NewMockServiceDiscoveryClient(mockController)

	// GetService from Cloudmap returns endpoint1 and endpoint2
	mock.EXPECT().GetService(gomock.Any(), test.NsName, test.SvcName).
		Return(test.GetTestService(), nil)
	// call to delete the endpoint not present in the k8s cluster
	mock.EXPECT().DeleteEndpoints(gomock.Any(), test.NsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint2()}).Return(nil).Times(1)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: test.NsName,
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
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.NsName, Name: test.SvcName}, serviceExport)
	assert.NoError(t, err)
	assert.Contains(t, serviceExport.Finalizers, ServiceExportFinalizer, "Finalizer added to the service export")
}

func TestServiceExportReconciler_Reconcile_DeleteExistingService(t *testing.T) {
	// create a fake controller client and add some objects
	serviceExportObj := testServiceExportObj()
	// Add finalizer string to the service
	serviceExportObj.Finalizers = []string{ServiceExportFinalizer}
	fakeClient := fake.NewClientBuilder().
		WithScheme(getServiceExportScheme()).
		WithObjects(serviceExportObj).
		WithLists(testEndpointSliceObj()).
		Build()

	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mock := cloudmap.NewMockServiceDiscoveryClient(mockController)

	// GetService from Cloudmap returns endpoint1 and endpoint2
	mock.EXPECT().GetService(gomock.Any(), test.NsName, test.SvcName).
		Return(test.GetTestService(), nil)
	// call to delete the endpoint in the cloudmap
	mock.EXPECT().DeleteEndpoints(gomock.Any(), test.NsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()}).Return(nil).Times(1)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: test.NsName,
			Name:      test.SvcName,
		},
	}

	reconciler := getServiceExportReconciler(t, mock, fakeClient)

	got, err := reconciler.Reconcile(context.Background(), request)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, got, "Result should be empty")

	serviceExport := &v1alpha1.ServiceExport{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.NsName, Name: test.SvcName}, serviceExport)
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

func getServiceExportReconciler(t *testing.T, mockClient *cloudmap.MockServiceDiscoveryClient, client client.Client) *ServiceExportReconciler {
	return &ServiceExportReconciler{
		Client:   client,
		Log:      common.NewLoggerWithLogr(testing2.TestLogger{T: t}),
		Scheme:   client.Scheme(),
		CloudMap: mockClient,
	}
}

func testEndpointSliceObj() *discovery.EndpointSliceList {
	port := int32(test.Port1)
	protocol := v1.ProtocolTCP
	endpointSlice := &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.NsName,
			Name:      test.SvcName + "-slice",
			Labels:    map[string]string{discovery.LabelServiceName: test.SvcName},
		},
		AddressType: discovery.AddressTypeIPv4,
		Endpoints: []discovery.Endpoint{{
			Addresses: []string{test.EndptIp1},
		}},
		Ports: []discovery.EndpointPort{{
			Name:     aws.String("http"),
			Protocol: &protocol,
			Port:     &port,
		}},
	}
	endpointSliceList := &discovery.EndpointSliceList{
		Items: []discovery.EndpointSlice{*endpointSlice},
	}
	return endpointSliceList
}

func testServiceObj() *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.SvcName,
			Namespace: test.NsName,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   test.Protocol1,
				Port:       test.ServicePort1,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: test.Port1},
			}},
		},
		Status: v1.ServiceStatus{},
	}
}

func testServiceExportObj() *v1alpha1.ServiceExport {
	return &v1alpha1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.SvcName,
			Namespace: test.NsName,
		},
	}
}
