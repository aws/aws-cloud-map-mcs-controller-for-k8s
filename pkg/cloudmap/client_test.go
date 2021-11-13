package cloudmap

import (
	"context"
	"errors"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	testing2 "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testSdClient struct {
	client    *serviceDiscoveryClient
	mockApi   cloudmap.MockServiceDiscoveryApi
	mockCache cloudmap.MockServiceDiscoveryClientCache
	close     func()
}

func TestNewServiceDiscoveryClient(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClient(&aws.Config{})
	assert.NotNil(t, sdc)
}

func TestServiceDiscoveryClient_ListServices_HappyCase(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(nil, false)
	tc.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{test.GetTestHttpNamespace()}, nil)
	tc.mockCache.EXPECT().CacheNamespace(test.GetTestHttpNamespace())

	tc.mockApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Name: test.SvcName, Id: test.SvcId}}, nil)
	tc.mockCache.EXPECT().CacheServiceId(test.NsName, test.SvcName, test.SvcId)

	tc.mockCache.EXPECT().GetEndpoints(test.NsName, test.SvcName).Return(nil, false)
	tc.mockApi.EXPECT().DiscoverInstances(context.TODO(), test.NsName, test.SvcName).
		Return([]types.HttpInstanceSummary{
			{
				InstanceId: aws.String(test.EndptId1),
				Attributes: map[string]string{
					model.EndpointIpv4Attr:      test.EndptIp1,
					model.EndpointPortAttr:      test.PortStr1,
					model.EndpointPortNameAttr:  test.PortName1,
					model.EndpointProtocolAttr:  test.Protocol1,
					model.ServicePortNameAttr:   test.PortName1,
					model.ServicePortAttr:       test.ServicePortStr1,
					model.ServiceProtocolAttr:   test.Protocol1,
					model.ServiceTargetPortAttr: test.PortStr1,
				},
			},
			{
				InstanceId: aws.String(test.EndptId2),
				Attributes: map[string]string{
					model.EndpointIpv4Attr:      test.EndptIp2,
					model.EndpointPortAttr:      test.PortStr2,
					model.EndpointPortNameAttr:  test.PortName2,
					model.EndpointProtocolAttr:  test.Protocol2,
					model.ServicePortNameAttr:   test.PortName2,
					model.ServicePortAttr:       test.ServicePortStr2,
					model.ServiceProtocolAttr:   test.Protocol2,
					model.ServiceTargetPortAttr: test.PortStr2,
				},
			},
		}, nil)
	tc.mockCache.EXPECT().CacheEndpoints(test.NsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})

	svcs, err := tc.client.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, []*model.Service{test.GetTestService()}, svcs)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_ListServices_HappyCaseCachedResults(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(test.GetTestHttpNamespace(), true)

	tc.mockApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Name: test.SvcName, Id: test.SvcId}}, nil)
	tc.mockCache.EXPECT().CacheServiceId(test.NsName, test.SvcName, test.SvcId)

	tc.mockCache.EXPECT().GetEndpoints(test.NsName, test.SvcName).
		Return([]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()}, true)

	svcs, err := tc.client.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, []*model.Service{test.GetTestService()}, svcs)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_ListServices_NamespaceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	nsErr := errors.New("error listing namespaces")
	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(nil, false)
	tc.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return(nil, nsErr)

	svcs, err := tc.client.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, nsErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_ServiceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(test.GetTestHttpNamespace(), true)

	svcErr := errors.New("error listing services")
	tc.mockApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{}, svcErr)

	svcs, err := tc.client.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, svcErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_InstanceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(test.GetTestHttpNamespace(), true)

	tc.mockApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Name: test.SvcName, Id: test.SvcId}}, nil)
	tc.mockCache.EXPECT().CacheServiceId(test.NsName, test.SvcName, test.SvcId)

	endptErr := errors.New("error listing endpoints")
	tc.mockCache.EXPECT().GetEndpoints(test.NsName, test.SvcName).Return(nil, false)
	tc.mockApi.EXPECT().DiscoverInstances(context.TODO(), test.NsName, test.SvcName).
		Return([]types.HttpInstanceSummary{}, endptErr)

	svcs, err := tc.client.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, endptErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_NamespaceNotFound(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(nil, false)
	tc.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)
	tc.mockCache.EXPECT().CacheNilNamespace(test.NsName)

	svcs, err := tc.client.ListServices(context.TODO(), test.NsName)
	assert.Empty(t, svcs)
	assert.Nil(t, err, "No error for namespace not found")
}

func TestServiceDiscoveryClient_CreateService_HappyCase(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(test.GetTestHttpNamespace(), true)

	tc.mockApi.EXPECT().CreateService(context.TODO(), *test.GetTestHttpNamespace(), test.SvcName).
		Return(test.SvcId, nil)
	tc.mockCache.EXPECT().CacheServiceId(test.NsName, test.SvcName, test.SvcId)

	err := tc.client.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_CreateService_HappyCaseForDNSNamespace(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(test.GetTestDnsNamespace(), true)

	tc.mockApi.EXPECT().CreateService(context.TODO(), *test.GetTestDnsNamespace(), test.SvcName).
		Return(test.SvcId, nil)
	tc.mockCache.EXPECT().CacheServiceId(test.NsName, test.SvcName, test.SvcId)

	err := tc.client.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_CreateService_NamespaceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	nsErr := errors.New("error listing namespaces")
	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(nil, false)
	tc.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nsErr)

	err := tc.client.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Equal(t, nsErr, err)
}

func TestServiceDiscoveryClient_CreateService_CreateServiceError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(test.GetTestDnsNamespace(), true)

	svcErr := errors.New("error creating service")
	tc.mockApi.EXPECT().CreateService(context.TODO(), *test.GetTestDnsNamespace(), test.SvcName).
		Return("", svcErr)

	err := tc.client.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Equal(t, err, svcErr)
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_HappyCase(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(nil, false)
	tc.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)
	tc.mockCache.EXPECT().CacheNilNamespace(test.NsName)

	tc.mockApi.EXPECT().CreateHttpNamespace(context.TODO(), test.NsName).
		Return(test.OpId1, nil)
	tc.mockApi.EXPECT().PollNamespaceOperation(context.TODO(), test.OpId1).
		Return(test.NsId, nil)
	tc.mockCache.EXPECT().CacheNamespace(test.GetTestHttpNamespace())

	tc.mockApi.EXPECT().CreateService(context.TODO(), *test.GetTestHttpNamespace(), test.SvcName).
		Return(test.SvcId, nil)
	tc.mockCache.EXPECT().CacheServiceId(test.NsName, test.SvcName, test.SvcId)

	err := tc.client.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_PollError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(nil, false)
	tc.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)
	tc.mockCache.EXPECT().CacheNilNamespace(test.NsName)

	pollErr := errors.New("polling error")
	tc.mockApi.EXPECT().CreateHttpNamespace(context.TODO(), test.NsName).
		Return(test.OpId1, nil)
	tc.mockApi.EXPECT().PollNamespaceOperation(context.TODO(), test.OpId1).
		Return("", pollErr)

	err := tc.client.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Equal(t, pollErr, err)
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_CreateNsError(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(nil, false)
	tc.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)
	tc.mockCache.EXPECT().CacheNilNamespace(test.NsName)

	nsErr := errors.New("create namespace error")
	tc.mockApi.EXPECT().CreateHttpNamespace(context.TODO(), test.NsName).
		Return("", nsErr)

	err := tc.client.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Equal(t, nsErr, err)
}

func TestServiceDiscoveryClient_GetService_HappyCase(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetEndpoints(test.NsName, test.SvcName).Return([]*model.Endpoint{}, false)

	tc.mockCache.EXPECT().GetServiceId(test.NsName, test.SvcName)

	tc.mockCache.EXPECT().GetNamespace(test.NsName).Return(nil, false)
	tc.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{test.GetTestHttpNamespace()}, nil)
	tc.mockCache.EXPECT().CacheNamespace(test.GetTestHttpNamespace())

	tc.mockApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Id: test.SvcId, Name: test.SvcName}}, nil)
	tc.mockCache.EXPECT().CacheServiceId(test.NsName, test.SvcName, test.SvcId)

	tc.mockCache.EXPECT().GetEndpoints(test.NsName, test.SvcName).Return([]*model.Endpoint{}, false)
	tc.mockApi.EXPECT().DiscoverInstances(context.TODO(), test.NsName, test.SvcName).
		Return([]types.HttpInstanceSummary{
			{
				InstanceId: aws.String(test.EndptId1),
				Attributes: map[string]string{
					model.EndpointIpv4Attr:      test.EndptIp1,
					model.EndpointPortAttr:      test.PortStr1,
					model.EndpointPortNameAttr:  test.PortName1,
					model.EndpointProtocolAttr:  test.Protocol1,
					model.ServicePortNameAttr:   test.PortName1,
					model.ServicePortAttr:       test.ServicePortStr1,
					model.ServiceProtocolAttr:   test.Protocol1,
					model.ServiceTargetPortAttr: test.PortStr1,
				},
			},
			{
				InstanceId: aws.String(test.EndptId2),
				Attributes: map[string]string{
					model.EndpointIpv4Attr:      test.EndptIp2,
					model.EndpointPortAttr:      test.PortStr2,
					model.EndpointPortNameAttr:  test.PortName2,
					model.EndpointProtocolAttr:  test.Protocol2,
					model.ServicePortNameAttr:   test.PortName2,
					model.ServicePortAttr:       test.ServicePortStr2,
					model.ServiceProtocolAttr:   test.Protocol2,
					model.ServiceTargetPortAttr: test.PortStr2,
				},
			},
		}, nil)
	tc.mockCache.EXPECT().CacheEndpoints(test.NsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})

	svc, err := tc.client.GetService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err)
	assert.Equal(t, test.GetTestService(), svc)
}

func TestServiceDiscoveryClient_GetService_CachedValues(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetEndpoints(test.NsName, test.SvcName).
		Return([]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()}, true)

	svc, err := tc.client.GetService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err)
	assert.Equal(t, test.GetTestService(), svc)
}

func TestServiceDiscoveryClient_RegisterEndpoints(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceId(test.NsName, test.SvcName).Return(test.SvcId, true)

	attrs1 := map[string]string{
		model.EndpointIpv4Attr:      test.EndptIp1,
		model.EndpointPortAttr:      test.PortStr1,
		model.EndpointPortNameAttr:  test.PortName1,
		model.EndpointProtocolAttr:  test.Protocol1,
		model.ServicePortNameAttr:   test.PortName1,
		model.ServicePortAttr:       test.ServicePortStr1,
		model.ServiceProtocolAttr:   test.Protocol1,
		model.ServiceTargetPortAttr: test.PortStr1,
	}
	attrs2 := map[string]string{
		model.EndpointIpv4Attr:      test.EndptIp2,
		model.EndpointPortAttr:      test.PortStr2,
		model.EndpointPortNameAttr:  test.PortName2,
		model.EndpointProtocolAttr:  test.Protocol2,
		model.ServicePortNameAttr:   test.PortName2,
		model.ServicePortAttr:       test.ServicePortStr2,
		model.ServiceProtocolAttr:   test.Protocol2,
		model.ServiceTargetPortAttr: test.PortStr2,
	}

	tc.mockApi.EXPECT().RegisterInstance(context.TODO(), test.SvcId, test.EndptId1, attrs1).
		Return(test.OpId1, nil)
	tc.mockApi.EXPECT().RegisterInstance(context.TODO(), test.SvcId, test.EndptId2, attrs2).
		Return(test.OpId2, nil)
	tc.mockApi.EXPECT().ListOperations(context.TODO(), gomock.Any()).
		Return(map[string]types.OperationStatus{
			test.OpId1: types.OperationStatusSuccess,
			test.OpId2: types.OperationStatusSuccess}, nil)

	tc.mockCache.EXPECT().EvictEndpoints(test.NsName, test.SvcName)

	err := tc.client.RegisterEndpoints(context.TODO(), test.NsName, test.SvcName,
		[]*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})

	assert.Nil(t, err)
}

func TestServiceDiscoveryClient_DeleteEndpoints(t *testing.T) {
	tc := getTestSdClient(t)
	defer tc.close()

	tc.mockCache.EXPECT().GetServiceId(test.NsName, test.SvcName).Return(test.SvcId, true)

	tc.mockApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1).Return(test.OpId1, nil)
	tc.mockApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId2).Return(test.OpId2, nil)
	tc.mockApi.EXPECT().ListOperations(context.TODO(), gomock.Any()).
		Return(map[string]types.OperationStatus{
			test.OpId1: types.OperationStatusSuccess,
			test.OpId2: types.OperationStatusSuccess}, nil)

	tc.mockCache.EXPECT().EvictEndpoints(test.NsName, test.SvcName)

	err := tc.client.DeleteEndpoints(context.TODO(), test.NsName, test.SvcName,
		[]*model.Endpoint{{Id: test.EndptId1}, {Id: test.EndptId2}})

	assert.Nil(t, err)
}

func getTestSdClient(t *testing.T) *testSdClient {
	mockController := gomock.NewController(t)
	mockCache := cloudmap.NewMockServiceDiscoveryClientCache(mockController)
	mockApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	return &testSdClient{
		client: &serviceDiscoveryClient{
			log:   testing2.TestLogger{T: t},
			sdApi: mockApi,
			cache: mockCache,
		},
		mockApi:   *mockApi,
		mockCache: *mockCache,
		close:     func() { mockController.Finish() },
	}
}
