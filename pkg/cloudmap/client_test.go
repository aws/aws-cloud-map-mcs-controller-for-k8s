package cloudmap

import (
	"context"
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cloudmapMock "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	aboutv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/about/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/go-logr/logr/testr"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testSdClient struct {
	client    *serviceDiscoveryClient
	mockApi   cloudmapMock.MockServiceDiscoveryApi
	mockCache cloudmapMock.MockServiceDiscoveryClientCache
	close     func()
}

func TestNewServiceDiscoveryClient(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()
	assert.NotNil(t, tc)
}

func TestServiceDiscoveryClient_ListServices_HappyCase(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceIdMap(test.HttpNsName).Return(nil, false)

	tc.mockCache.EXPECT().GetNamespaceMap().Return(nil, false)
	tc.mockApi.EXPECT().GetNamespaceMap(context.TODO()).Return(getNamespaceMapForTest(), nil)
	tc.mockCache.EXPECT().CacheNamespaceMap(getNamespaceMapForTest())

	tc.mockApi.EXPECT().GetServiceIdMap(context.TODO(), test.HttpNsId).Return(getServiceIdMapForTest(), nil)
	tc.mockCache.EXPECT().CacheServiceIdMap(test.HttpNsName, getServiceIdMapForTest())

	tc.mockCache.EXPECT().GetEndpoints(test.HttpNsName, test.SvcName).Return(nil, false)
	tc.mockApi.EXPECT().DiscoverInstances(context.TODO(), test.HttpNsName, test.SvcName, &map[string]string{
		model.ClusterSetIdAttr: test.ClusterSet,
	}).Return(getHttpInstanceSummaryForTest(), nil)

	tc.mockCache.EXPECT().CacheEndpoints(test.HttpNsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})

	svcs, err := tc.client.ListServices(context.TODO(), test.HttpNsName)
	assert.Equal(t, []*model.Service{test.GetTestService()}, svcs)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_ListServices_HappyCaseCachedResults(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	dnsService := test.GetTestService()
	dnsService.Namespace = test.DnsNsName

	tc.mockCache.EXPECT().GetServiceIdMap(test.DnsNsName).Return(getServiceIdMapForTest(), true)

	tc.mockCache.EXPECT().GetEndpoints(test.DnsNsName, test.SvcName).
		Return([]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()}, true)

	svcs, err := tc.client.ListServices(context.TODO(), test.DnsNsName)
	assert.Equal(t, []*model.Service{dnsService}, svcs)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_ListServices_NamespaceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceIdMap(test.HttpNsName).Return(nil, false)

	nsErr := errors.New("error listing namespaces")
	tc.mockCache.EXPECT().GetNamespaceMap().Return(nil, false)
	tc.mockApi.EXPECT().GetNamespaceMap(context.TODO()).Return(nil, nsErr)

	svcs, err := tc.client.ListServices(context.TODO(), test.HttpNsName)
	assert.Equal(t, nsErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_ServiceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceIdMap(test.HttpNsName).Return(nil, false)

	tc.mockCache.EXPECT().GetNamespaceMap().Return(getNamespaceMapForTest(), true)

	svcErr := errors.New("error listing services")
	tc.mockApi.EXPECT().GetServiceIdMap(context.TODO(), test.HttpNsId).
		Return(nil, svcErr)

	svcs, err := tc.client.ListServices(context.TODO(), test.HttpNsName)
	assert.Equal(t, svcErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_InstanceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceIdMap(test.HttpNsName).Return(getServiceIdMapForTest(), true)

	endptErr := errors.New("error listing endpoints")
	tc.mockCache.EXPECT().GetEndpoints(test.HttpNsName, test.SvcName).Return(nil, false)
	tc.mockApi.EXPECT().DiscoverInstances(context.TODO(), test.HttpNsName, test.SvcName, &map[string]string{
		model.ClusterSetIdAttr: test.ClusterSet,
	}).
		Return([]types.HttpInstanceSummary{}, endptErr)

	svcs, err := tc.client.ListServices(context.TODO(), test.HttpNsName)
	assert.Equal(t, endptErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_NamespaceNotFound(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceIdMap(test.HttpNsName).Return(nil, false)
	tc.mockCache.EXPECT().GetNamespaceMap().Return(nil, true)

	svcs, err := tc.client.ListServices(context.TODO(), test.HttpNsName)
	assert.Empty(t, svcs)
	assert.Nil(t, err, "No error for namespace not found")
}

func TestServiceDiscoveryClient_CreateService_HappyCase(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespaceMap().Return(getNamespaceMapForTest(), true)

	tc.mockApi.EXPECT().CreateService(context.TODO(), *test.GetTestHttpNamespace(), test.SvcName).
		Return(test.SvcId, nil)
	tc.mockCache.EXPECT().EvictServiceIdMap(test.HttpNsName)

	err := tc.client.CreateService(context.TODO(), test.HttpNsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_CreateService_HappyCaseForDNSNamespace(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespaceMap().Return(getNamespaceMapForTest(), true)

	tc.mockApi.EXPECT().CreateService(context.TODO(), *test.GetTestDnsNamespace(), test.SvcName).
		Return(test.SvcId, nil)
	tc.mockCache.EXPECT().EvictServiceIdMap(test.DnsNsName)

	err := tc.client.CreateService(context.TODO(), test.DnsNsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_CreateService_NamespaceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	nsErr := errors.New("error listing namespaces")

	tc.mockCache.EXPECT().GetNamespaceMap().Return(nil, false)
	tc.mockApi.EXPECT().GetNamespaceMap(context.TODO()).Return(nil, nsErr)

	err := tc.client.CreateService(context.TODO(), test.HttpNsName, test.SvcName)
	assert.Equal(t, nsErr, err)
}

func TestServiceDiscoveryClient_CreateService_CreateServiceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespaceMap().Return(getNamespaceMapForTest(), true)

	svcErr := errors.New("error creating service")
	tc.mockApi.EXPECT().CreateService(context.TODO(), *test.GetTestDnsNamespace(), test.SvcName).
		Return("", svcErr)

	err := tc.client.CreateService(context.TODO(), test.DnsNsName, test.SvcName)
	assert.Equal(t, err, svcErr)
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_HappyCase(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespaceMap().Return(map[string]*model.Namespace{
		test.DnsNsName: test.GetTestDnsNamespace(),
	}, true)

	tc.mockApi.EXPECT().CreateHttpNamespace(context.TODO(), test.HttpNsName).
		Return(test.OpId1, nil)
	tc.mockApi.EXPECT().PollNamespaceOperation(context.TODO(), test.OpId1).
		Return(test.HttpNsId, nil)
	tc.mockCache.EXPECT().EvictNamespaceMap()

	tc.mockApi.EXPECT().CreateService(context.TODO(), *test.GetTestHttpNamespace(), test.SvcName).
		Return(test.SvcId, nil)
	tc.mockCache.EXPECT().EvictServiceIdMap(test.HttpNsName)

	err := tc.client.CreateService(context.TODO(), test.HttpNsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_PollError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespaceMap().Return(nil, true)

	pollErr := errors.New("polling error")
	tc.mockApi.EXPECT().CreateHttpNamespace(context.TODO(), test.HttpNsName).
		Return(test.OpId1, nil)
	tc.mockApi.EXPECT().PollNamespaceOperation(context.TODO(), test.OpId1).
		Return("", pollErr)

	err := tc.client.CreateService(context.TODO(), test.HttpNsName, test.SvcName)
	assert.Equal(t, pollErr, err)
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_CreateNsError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespaceMap().Return(nil, true)

	nsErr := errors.New("create namespace error")
	tc.mockApi.EXPECT().CreateHttpNamespace(context.TODO(), test.HttpNsName).
		Return("", nsErr)

	err := tc.client.CreateService(context.TODO(), test.HttpNsName, test.SvcName)
	assert.Equal(t, nsErr, err)
}

func TestServiceDiscoveryClient_GetService_HappyCase(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetEndpoints(test.HttpNsName, test.SvcName).Return(nil, false)

	tc.mockCache.EXPECT().GetServiceIdMap(test.HttpNsName).Return(nil, false)

	tc.mockCache.EXPECT().GetNamespaceMap().Return(nil, false)
	tc.mockApi.EXPECT().GetNamespaceMap(context.TODO()).
		Return(getNamespaceMapForTest(), nil)
	tc.mockCache.EXPECT().CacheNamespaceMap(getNamespaceMapForTest())

	tc.mockApi.EXPECT().GetServiceIdMap(context.TODO(), test.HttpNsId).
		Return(map[string]string{test.SvcName: test.SvcId}, nil)
	tc.mockCache.EXPECT().CacheServiceIdMap(test.HttpNsName, getServiceIdMapForTest())

	tc.mockCache.EXPECT().GetEndpoints(test.HttpNsName, test.SvcName).Return([]*model.Endpoint{}, false)
	tc.mockApi.EXPECT().DiscoverInstances(context.TODO(), test.HttpNsName, test.SvcName, &map[string]string{
		model.ClusterSetIdAttr: test.ClusterSet,
	}).
		Return(getHttpInstanceSummaryForTest(), nil)
	tc.mockCache.EXPECT().CacheEndpoints(test.HttpNsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})

	svc, err := tc.client.GetService(context.TODO(), test.HttpNsName, test.SvcName)
	assert.Nil(t, err)
	assert.Equal(t, test.GetTestService(), svc)
}

func TestServiceDiscoveryClient_GetService_CachedValues(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetEndpoints(test.HttpNsName, test.SvcName).
		Return([]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()}, true)

	svc, err := tc.client.GetService(context.TODO(), test.HttpNsName, test.SvcName)
	assert.Nil(t, err)
	assert.Equal(t, test.GetTestService(), svc)
}

func TestServiceDiscoveryClient_RegisterEndpoints(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceIdMap(test.HttpNsName).Return(getServiceIdMapForTest(), true)

	attrs1 := map[string]string{
		model.ClusterIdAttr:             test.ClusterId1,
		model.ClusterSetIdAttr:          test.ClusterSet,
		model.EndpointIpv4Attr:          test.EndptIp1,
		model.EndpointPortAttr:          test.PortStr1,
		model.EndpointPortNameAttr:      test.PortName1,
		model.EndpointProtocolAttr:      test.Protocol1,
		model.EndpointReadyAttr:         test.EndptReadyTrue,
		model.ServicePortNameAttr:       test.PortName1,
		model.ServicePortAttr:           test.ServicePortStr1,
		model.ServiceProtocolAttr:       test.Protocol1,
		model.ServiceTargetPortAttr:     test.PortStr1,
		model.ServiceTypeAttr:           test.SvcType,
		model.EndpointHostnameAttr:      test.Hostname,
		model.EndpointNodeNameAttr:      test.Nodename,
		model.ServiceExportCreationAttr: strconv.FormatInt(test.SvcExportCreationTimestamp, 10),
		model.K8sVersionAttr:            test.PackageVersion,
	}
	attrs2 := map[string]string{
		model.ClusterIdAttr:             test.ClusterId1,
		model.ClusterSetIdAttr:          test.ClusterSet,
		model.EndpointIpv4Attr:          test.EndptIp2,
		model.EndpointPortAttr:          test.PortStr2,
		model.EndpointPortNameAttr:      test.PortName2,
		model.EndpointProtocolAttr:      test.Protocol2,
		model.EndpointReadyAttr:         test.EndptReadyTrue,
		model.ServicePortNameAttr:       test.PortName2,
		model.ServicePortAttr:           test.ServicePortStr2,
		model.ServiceProtocolAttr:       test.Protocol2,
		model.ServiceTargetPortAttr:     test.PortStr2,
		model.ServiceTypeAttr:           test.SvcType,
		model.EndpointHostnameAttr:      test.Hostname,
		model.EndpointNodeNameAttr:      test.Nodename,
		model.ServiceExportCreationAttr: strconv.FormatInt(test.SvcExportCreationTimestamp, 10),
		model.K8sVersionAttr:            test.PackageVersion,
	}

	tc.mockApi.EXPECT().RegisterInstance(context.TODO(), test.SvcId, test.EndptId1, attrs1).
		Return(test.OpId1, nil)
	tc.mockApi.EXPECT().RegisterInstance(context.TODO(), test.SvcId, test.EndptId2, attrs2).
		Return(test.OpId2, nil)
	tc.mockApi.EXPECT().ListOperations(context.TODO(), gomock.Any()).
		Return(map[string]types.OperationStatus{
			test.OpId1: types.OperationStatusSuccess,
			test.OpId2: types.OperationStatusSuccess}, nil)

	tc.mockCache.EXPECT().EvictEndpoints(test.HttpNsName, test.SvcName)

	err := tc.client.RegisterEndpoints(context.TODO(), test.HttpNsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})

	assert.Nil(t, err)
}

func TestServiceDiscoveryClient_DeleteEndpoints(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceIdMap(test.HttpNsName).Return(getServiceIdMapForTest(), true)

	tc.mockApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1).Return(test.OpId1, nil)
	tc.mockApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId2).Return(test.OpId2, nil)
	tc.mockApi.EXPECT().ListOperations(context.TODO(), gomock.Any()).
		Return(map[string]types.OperationStatus{
			test.OpId1: types.OperationStatusSuccess,
			test.OpId2: types.OperationStatusSuccess}, nil)

	tc.mockCache.EXPECT().EvictEndpoints(test.HttpNsName, test.SvcName)

	err := tc.client.DeleteEndpoints(context.TODO(), test.HttpNsName, test.SvcName,
		[]*model.Endpoint{
			{Id: test.EndptId1, ClusterId: test.ClusterId1, ClusterSetId: test.ClusterSet},
			{Id: test.EndptId2, ClusterId: test.ClusterId1, ClusterSetId: test.ClusterSet},
		})
	assert.Nil(t, err)
}

func getTestSdClient(t *testing.T) *testSdClient {
	test.SetTestVersion()
	mockController := gomock.NewController(t)
	mockCache := cloudmapMock.NewMockServiceDiscoveryClientCache(mockController)
	mockApi := cloudmapMock.NewMockServiceDiscoveryApi(mockController)
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(aboutv1alpha1.GroupVersion, &aboutv1alpha1.ClusterProperty{}, &aboutv1alpha1.ClusterPropertyList{})
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(test.ClusterIdForTest(), test.ClusterSetIdForTest()).Build()
	return &testSdClient{
		client: &serviceDiscoveryClient{
			log:          common.NewLoggerWithLogr(testr.New(t)),
			sdApi:        mockApi,
			cache:        mockCache,
			clusterUtils: model.NewClusterUtils(fakeClient),
		},
		mockApi:   *mockApi,
		mockCache: *mockCache,
		close:     func() { mockController.Finish() },
	}
}

func getHttpInstanceSummaryForTest() []types.HttpInstanceSummary {
	return []types.HttpInstanceSummary{
		{
			InstanceId: aws.String(test.EndptId1),
			Attributes: map[string]string{
				model.ClusterIdAttr:             test.ClusterId1,
				model.ClusterSetIdAttr:          test.ClusterSet,
				model.EndpointIpv4Attr:          test.EndptIp1,
				model.EndpointPortAttr:          test.PortStr1,
				model.EndpointPortNameAttr:      test.PortName1,
				model.EndpointProtocolAttr:      test.Protocol1,
				model.EndpointReadyAttr:         test.EndptReadyTrue,
				model.ServicePortNameAttr:       test.PortName1,
				model.ServicePortAttr:           test.ServicePortStr1,
				model.ServiceProtocolAttr:       test.Protocol1,
				model.ServiceTargetPortAttr:     test.PortStr1,
				model.ServiceTypeAttr:           test.SvcType,
				model.EndpointHostnameAttr:      test.Hostname,
				model.EndpointNodeNameAttr:      test.Nodename,
				model.ServiceExportCreationAttr: strconv.FormatInt(test.SvcExportCreationTimestamp, 10),
				model.K8sVersionAttr:            test.PackageVersion,
			},
		},
		{
			InstanceId: aws.String(test.EndptId2),
			Attributes: map[string]string{
				model.ClusterIdAttr:             test.ClusterId1,
				model.ClusterSetIdAttr:          test.ClusterSet,
				model.EndpointIpv4Attr:          test.EndptIp2,
				model.EndpointPortAttr:          test.PortStr2,
				model.EndpointPortNameAttr:      test.PortName2,
				model.EndpointProtocolAttr:      test.Protocol2,
				model.EndpointReadyAttr:         test.EndptReadyTrue,
				model.ServicePortNameAttr:       test.PortName2,
				model.ServicePortAttr:           test.ServicePortStr2,
				model.ServiceProtocolAttr:       test.Protocol2,
				model.ServiceTargetPortAttr:     test.PortStr2,
				model.ServiceTypeAttr:           test.SvcType,
				model.EndpointHostnameAttr:      test.Hostname,
				model.EndpointNodeNameAttr:      test.Nodename,
				model.ServiceExportCreationAttr: strconv.FormatInt(test.SvcExportCreationTimestamp, 10),
				model.K8sVersionAttr:            test.PackageVersion,
			},
		},
	}
}

func getNamespaceMapForTest() map[string]*model.Namespace {
	return map[string]*model.Namespace{
		test.HttpNsName: test.GetTestHttpNamespace(),
		test.DnsNsName:  test.GetTestDnsNamespace(),
	}
}

func getServiceIdMapForTest() map[string]string {
	return map[string]string{test.SvcName: test.SvcId}
}
