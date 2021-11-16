package controllers

import (
	"context"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/api/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	testingLogger "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

func TestCloudMapReconciler_Reconcile(t *testing.T) {
	// create a fake controller client and add some objects
	objs := []runtime.Object{testNamespace()}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.ServiceImportList{}, &v1alpha1.ServiceImport{})

	fakeClient := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()

	// create a mock cloudmap service discovery client
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mockSDClient := cloudmap.NewMockServiceDiscoveryClient(mockController)
	// The service model in the Cloudmap
	mockSDClient.EXPECT().ListServices(context.TODO(), test.NsName).
		Return([]*model.Service{test.GetTestServiceWithEndpoint1()}, nil)

	reconciler := getReconciler(t, mockSDClient, fakeClient)

	err := reconciler.Reconcile(context.TODO())
	if err != nil {
		t.Fatalf("reconcile failed: (%v)", err)
	}

	// assert service import object
	serviceImport := &v1alpha1.ServiceImport{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.NsName, Name: test.SvcName}, serviceImport)
	assert.NoError(t, err)
	assert.Equal(t, test.SvcName, serviceImport.Name, "Service imported")

	// assert derived service is successfully created
	derivedServiceList := &v1.ServiceList{}
	err = fakeClient.List(context.TODO(), derivedServiceList, client.InNamespace(test.NsName))
	assert.NoError(t, err)
	derivedService := derivedServiceList.Items[0]
	assert.True(t, strings.Contains(derivedService.Name, "imported-"), "Derived service created", "service", derivedService.Name)
	assert.Equal(t, int32(test.ServicePort1), derivedService.Spec.Ports[0].Port)
	assert.Equal(t, int32(test.Port1), derivedService.Spec.Ports[0].TargetPort.IntVal)

	// assert endpoint slices are created
	endpointSliceList := &v1beta1.EndpointSliceList{}
	err = fakeClient.List(context.TODO(), endpointSliceList, client.InNamespace(test.NsName))
	assert.NoError(t, err)
	endpointSlice := endpointSliceList.Items[0]
	assert.Equal(t, test.SvcName, endpointSlice.Labels["multicluster.kubernetes.io/service-name"], "Endpoint slice is created")
	assert.Equal(t, int32(test.Port1), *endpointSlice.Ports[0].Port)
	assert.Equal(t, test.EndptIp1, endpointSlice.Endpoints[0].Addresses[0])
}

func testNamespace() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.NsName,
			Namespace: test.NsName,
		},
	}
}

func getReconciler(t *testing.T, mockSDClient *cloudmap.MockServiceDiscoveryClient, client client.Client) *CloudMapReconciler {
	return &CloudMapReconciler{
		Client:   client,
		Cloudmap: mockSDClient,
		Logger:   testingLogger.TestLogger{T: t},
	}
}
