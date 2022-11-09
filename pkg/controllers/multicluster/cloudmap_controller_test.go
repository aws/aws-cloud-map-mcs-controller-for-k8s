package controllers

import (
	"context"
	"strings"
	"testing"

	aboutv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/about/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	cloudmapMock "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	multiclusterv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/go-logr/logr/testr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCloudMapReconciler_Reconcile(t *testing.T) {
	// create a fake controller client and add some objects
	svcImportToBeDeleted := serviceImportForTest("svc1")
	fakeClient := fake.NewClientBuilder().WithScheme(getCloudMapReconcilerScheme()).
		WithObjects(k8sNamespaceForTest(), serviceImportForTest(test.SvcName), svcImportToBeDeleted,
			test.ClusterIdForTest(), test.ClusterSetIdForTest()).Build()

	// create a mock cloudmap service discovery client
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mockSDClient := cloudmapMock.NewMockServiceDiscoveryClient(mockController)
	// The service model in the Cloudmap
	mockSDClient.EXPECT().ListServices(context.TODO(), test.HttpNsName).
		Return([]*model.Service{test.GetTestServiceWithEndpoint([]*model.Endpoint{test.GetTestEndpoint1()})}, nil)

	reconciler := getReconciler(t, mockSDClient, fakeClient)

	err := reconciler.Reconcile(context.TODO())
	if err != nil {
		t.Fatalf("reconcile failed: (%v)", err)
	}

	// assert service import object
	serviceImport := &multiclusterv1alpha1.ServiceImport{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.HttpNsName, Name: test.SvcName}, serviceImport)
	assert.NoError(t, err)
	assert.Equal(t, test.SvcName, serviceImport.Name, "Service imported")

	// assert derived service is successfully created
	derivedServiceList := &v1.ServiceList{}
	err = fakeClient.List(context.TODO(), derivedServiceList, client.InNamespace(test.HttpNsName))
	assert.NoError(t, err)
	derivedService := derivedServiceList.Items[0]
	assertDerivedService(t, &derivedService, test.ServicePort1, test.Port1)

	// assert endpoint slices are created
	endpointSliceList := &discovery.EndpointSliceList{}
	err = fakeClient.List(context.TODO(), endpointSliceList, client.InNamespace(test.HttpNsName))
	assert.NoError(t, err)
	endpointSlice := endpointSliceList.Items[0]
	assertEndpointSlice(t, &endpointSlice, test.Port1, test.EndptIp1, test.ClusterId1)

	// assert svcImportToBeDeleted is not found in list
	serviceImports := &multiclusterv1alpha1.ServiceImportList{}
	err = fakeClient.List(context.TODO(), serviceImports, client.InNamespace(test.HttpNsName))
	assert.NoError(t, err)
	assert.True(t, len(serviceImports.Items) == 1)
	assert.Equal(t, serviceImports.Items[0].Name, test.SvcName)
}

func TestCloudMapReconciler_Reconcile_MulticlusterService(t *testing.T) {
	// create a fake controller client and add some objects
	fakeClient := fake.NewClientBuilder().WithScheme(getCloudMapReconcilerScheme()).
		WithObjects(k8sNamespaceForTest(), test.ClusterIdForTest(), test.ClusterSetIdForTest()).Build()

	// create a mock cloudmap service discovery client
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mockSDClient := cloudmapMock.NewMockServiceDiscoveryClient(mockController)
	// The service model in the Cloudmap.
	mockSDClient.EXPECT().ListServices(context.TODO(), test.HttpNsName).
		// The multicluster service has endpoints in different clusters (different ClusterIds)
		Return([]*model.Service{test.GetTestMulticlusterService()}, nil)

	reconciler := getReconciler(t, mockSDClient, fakeClient)

	err := reconciler.Reconcile(context.TODO())
	if err != nil {
		t.Fatalf("reconcile failed: (%v)", err)
	}

	// assert service import object
	svcImport := &multiclusterv1alpha1.ServiceImport{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: test.HttpNsName, Name: test.SvcName}, svcImport)
	assert.NoError(t, err)
	assert.Equal(t, test.SvcName, svcImport.Name, "Service imported")

	assert.Contains(t, svcImport.Status.Clusters, multiclusterv1alpha1.ClusterStatus{Cluster: test.ClusterId1})
	assert.Contains(t, svcImport.Status.Clusters, multiclusterv1alpha1.ClusterStatus{Cluster: test.ClusterId2})
	assert.Equal(t, 2, len(svcImport.Status.Clusters))

	// assert derived services are successfully created
	derivedServiceList := &v1.ServiceList{}
	err = fakeClient.List(context.TODO(), derivedServiceList, client.InNamespace(test.HttpNsName))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(derivedServiceList.Items))

	derivedServiceMap := map[string]v1.Service{}
	for _, derivedService := range derivedServiceList.Items {
		derivedServiceMap[derivedService.ObjectMeta.Name] = derivedService
	}

	derivedService1 := derivedServiceMap[DerivedName(svcImport.Namespace, svcImport.Name, test.ClusterId1)]
	assertDerivedService(t, &derivedService1, test.ServicePort1, test.Port1)
	derivedService2 := derivedServiceMap[DerivedName(svcImport.Namespace, svcImport.Name, test.ClusterId2)]
	assertDerivedService(t, &derivedService2, test.ServicePort2, test.Port2)

	// assert endpoint slices are created for each derived service
	endpointSliceList := &discovery.EndpointSliceList{}
	err = fakeClient.List(context.TODO(), endpointSliceList, client.InNamespace(test.HttpNsName))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(endpointSliceList.Items))

	endpointSliceMap := make(map[string]discovery.EndpointSlice)
	for _, endpointSlice := range endpointSliceList.Items {
		endpointSliceName := endpointSlice.ObjectMeta.Name
		derivedServiceName := endpointSliceName[:strings.LastIndex(endpointSliceName, "-")]
		endpointSliceMap[derivedServiceName] = endpointSlice
	}

	endpointSlice1 := endpointSliceMap[derivedService1.Name]
	assertEndpointSlice(t, &endpointSlice1, test.Port1, test.EndptIp1, test.ClusterId1)
	endpointSlice2 := endpointSliceMap[derivedService2.Name]
	assertEndpointSlice(t, &endpointSlice2, test.Port2, test.EndptIp2, test.ClusterId2)
}

func getCloudMapReconcilerScheme() *runtime.Scheme {
	s := scheme.Scheme
	s.AddKnownTypes(multiclusterv1alpha1.GroupVersion, &multiclusterv1alpha1.ServiceImportList{}, &multiclusterv1alpha1.ServiceImport{})
	s.AddKnownTypes(aboutv1alpha1.GroupVersion, &aboutv1alpha1.ClusterProperty{}, &aboutv1alpha1.ClusterPropertyList{})
	return s
}

func getReconciler(t *testing.T, mockSDClient *cloudmapMock.MockServiceDiscoveryClient, client client.Client) *CloudMapReconciler {
	return &CloudMapReconciler{
		Client:       client,
		Cloudmap:     mockSDClient,
		Log:          common.NewLoggerWithLogr(testr.New(t)),
		ClusterUtils: model.NewClusterUtils(client),
	}
}

func assertDerivedService(t *testing.T, derivedService *v1.Service, servicePort int, port int) {
	assert.NotNil(t, derivedService)
	assert.True(t, strings.Contains(derivedService.Name, "imported-"), "Derived service created", "service", derivedService.Name)
	assert.Equal(t, int32(servicePort), derivedService.Spec.Ports[0].Port)
	assert.Equal(t, int32(port), derivedService.Spec.Ports[0].TargetPort.IntVal)
}

func assertEndpointSlice(t *testing.T, endpointSlice *discovery.EndpointSlice, port int, endptIp string, clusterId string) {
	assert.NotNil(t, endpointSlice)
	assert.Equal(t, test.SvcName, endpointSlice.Labels["multicluster.kubernetes.io/service-name"], "Endpoint slice is created")
	assert.Equal(t, clusterId, endpointSlice.Labels["multicluster.kubernetes.io/source-cluster"], "Endpoint slice is created")
	assert.Contains(t, endpointSlice.Labels, LabelEndpointSliceManagedBy, "Managed by label is added")
	assert.Equal(t, int32(port), *endpointSlice.Ports[0].Port)
	assert.Equal(t, 1, len(endpointSlice.Endpoints))
	assert.Equal(t, endptIp, endpointSlice.Endpoints[0].Addresses[0])
}
