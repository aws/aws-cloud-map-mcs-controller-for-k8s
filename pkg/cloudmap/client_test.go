package cloudmap

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	testing2 "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/cache"
	"testing"
	"time"
)

func TestNewServiceDiscoveryClient(t *testing.T) {
	sdc := NewServiceDiscoveryClient(&aws.Config{})
	assert.NotNil(t, sdc)
}

func TestServiceDiscoveryClient_ListServices_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{test.GetTestHttpNamespace()}, nil)
	sdApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Name: test.SvcName, Id: test.SvcId}}, nil)
	sdApi.EXPECT().ListInstances(context.TODO(), test.SvcId).
		Return([]types.InstanceSummary{
			{
				Id: aws.String(test.EndptId1),
				Attributes: map[string]string{
					model.Ipv4Attr: test.EndptIp1,
					model.PortAttr: test.EndptPortStr1,
				},
			},
			{
				Id: aws.String(test.EndptId2),
				Attributes: map[string]string{
					model.Ipv4Attr: test.EndptIp2,
					model.PortAttr: test.EndptPortStr2,
				},
			},
		}, nil)

	sdc := getTestSdClient(t, sdApi)
	svcs, err := sdc.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, []*model.Service{test.GetTestService()}, svcs)
	assert.Nil(t, err, "No error for happy case")

	cachedNs, _ := sdc.namespaceCache.Get(test.NsName)
	assert.Equal(t, *test.GetTestHttpNamespace(), cachedNs, "Happy case caches namespace ID")
	cachedSvc, _ := sdc.serviceIdCache.Get(fmt.Sprintf("%s/%s", test.NsName, test.SvcName))
	assert.Equal(t, test.SvcId, cachedSvc, "Happy case caches service ID")
	cachedEndpts, _ := sdc.endpointCache.Get(test.SvcId)
	assert.Equal(t, []*model.Endpoint{test.GetTestEndpoint(), test.GetTestEndpoint2()}, cachedEndpts, "Happy case caches endpoints")
}

func TestServiceDiscoveryClient_ListServices_HappyCaseCachedResults(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Name: test.SvcName, Id: test.SvcId}}, nil)

	sdc := getTestSdClient(t, sdApi)
	sdc.namespaceCache.Add(test.NsName, *test.GetTestHttpNamespace(), time.Minute)
	sdc.endpointCache.Add(test.SvcId, []*model.Endpoint{test.GetTestEndpoint(), test.GetTestEndpoint2()}, time.Minute)

	svcs, err := sdc.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, []*model.Service{test.GetTestService()}, svcs)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_ListServices_NamespaceError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	nsErr := errors.New("error listing namespaces")
	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nsErr)

	sdc := getTestSdClient(t, sdApi)
	svcs, err := sdc.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, nsErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_ServiceError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	svcErr := errors.New("error listing services")

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{test.GetTestHttpNamespace()}, nil)
	sdApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{}, svcErr)

	sdc := getTestSdClient(t, sdApi)
	svcs, err := sdc.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, svcErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_InstanceError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	endptErr := errors.New("error listing endpoints")
	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{test.GetTestHttpNamespace()}, nil)
	sdApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Name: test.SvcName, Id: test.SvcId}}, nil)
	sdApi.EXPECT().ListInstances(context.TODO(), test.SvcId).
		Return([]types.InstanceSummary{}, endptErr)

	sdc := getTestSdClient(t, sdApi)
	svcs, err := sdc.ListServices(context.TODO(), test.NsName)
	assert.Equal(t, endptErr, err)
	assert.Empty(t, svcs)
}

func TestServiceDiscoveryClient_ListServices_NamespaceNotFound(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)

	sdc := getTestSdClient(t, sdApi)
	svcs, err := sdc.ListServices(context.TODO(), test.NsName)
	assert.Empty(t, svcs)
	assert.Nil(t, err, "No error for namespace not found")

	cachedNs, found := sdc.namespaceCache.Get(test.NsName)
	assert.True(t, found)
	assert.Nil(t, cachedNs, "Namespace not found in the cache")
}

func TestServiceDiscoveryClient_CreateService_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{test.GetTestHttpNamespace()}, nil)
	sdApi.EXPECT().CreateService(context.TODO(), *test.GetTestHttpNamespace(), test.SvcName).
		Return(test.SvcId, nil)

	sdc := getTestSdClient(t, sdApi)
	err := sdc.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")

	cachedNs, _ := sdc.namespaceCache.Get(test.NsName)
	assert.Equal(t, *test.GetTestHttpNamespace(), cachedNs, "Happy case caches namespace")
	cachedSvc, _ := sdc.serviceIdCache.Get(fmt.Sprintf("%s/%s", test.NsName, test.SvcName))
	assert.Equal(t, test.SvcId, cachedSvc, "Happy case caches service ID")
}

func TestServiceDiscoveryClient_CreateService_HappyCaseForDNSNamespace(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{test.GetTestDnsNamespace()}, nil)
	sdApi.EXPECT().CreateService(context.TODO(), *test.GetTestDnsNamespace(), test.SvcName).
		Return(test.SvcId, nil)

	sdc := getTestSdClient(t, sdApi)
	err := sdc.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")

	cachedNs, _ := sdc.namespaceCache.Get(test.NsName)
	assert.Equal(t, *test.GetTestDnsNamespace(), cachedNs, "Happy case caches namespace")
	cachedSvc, _ := sdc.serviceIdCache.Get(fmt.Sprintf("%s/%s", test.NsName, test.SvcName))
	assert.Equal(t, test.SvcId, cachedSvc, "Happy case caches service ID")
}

func TestServiceDiscoveryClient_CreateService_HappyCaseCachedResults(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().CreateService(context.TODO(), *test.GetTestHttpNamespace(), test.SvcName).
		Return(test.SvcId, nil)

	sdc := getTestSdClient(t, sdApi)
	sdc.namespaceCache.Add(test.NsName, *test.GetTestHttpNamespace(), time.Minute)

	err := sdc.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")
}

func TestServiceDiscoveryClient_CreateService_NamespaceError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	nsErr := errors.New("error listing namespaces")
	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nsErr)

	sdc := getTestSdClient(t, sdApi)
	err := sdc.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Equal(t, nsErr, err)
}

func TestServiceDiscoveryClient_CreateService_CreateServiceError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	svcErr := errors.New("error creating service")
	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{test.GetTestHttpNamespace()}, nil)
	sdApi.EXPECT().CreateService(context.TODO(), *test.GetTestHttpNamespace(), test.SvcName).
		Return("", svcErr)

	sdc := getTestSdClient(t, sdApi)
	err := sdc.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Equal(t, err, svcErr)
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdc := getTestSdClient(t, sdApi)

	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)
	sdApi.EXPECT().CreateHttpNamespace(context.TODO(), test.NsName).
		Return(test.OpId1, nil)
	sdApi.EXPECT().PollNamespaceOperation(context.TODO(), test.OpId1).
		Return(test.NsId, nil)
	sdApi.EXPECT().CreateService(context.TODO(), *test.GetTestHttpNamespace(), test.SvcName).
		Return(test.SvcId, nil)

	err := sdc.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")

	cachedNs, _ := sdc.namespaceCache.Get(test.NsName)
	assert.Equal(t, *test.GetTestHttpNamespace(), cachedNs, "Create namespace caches namespace ID")
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_PollError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	pollErr := errors.New("polling error")
	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)
	sdApi.EXPECT().CreateHttpNamespace(context.TODO(), test.NsName).
		Return(test.OpId1, nil)
	sdApi.EXPECT().PollNamespaceOperation(context.TODO(), test.OpId1).
		Return("", pollErr)

	sdc := getTestSdClient(t, sdApi)
	err := sdc.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Equal(t, pollErr, err)
}

func TestServiceDiscoveryClient_CreateService_CreatesNamespace_CreateNsError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	nsErr := errors.New("create namespace error")
	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)
	sdApi.EXPECT().CreateHttpNamespace(context.TODO(), test.NsName).
		Return("", nsErr)

	sdc := getTestSdClient(t, sdApi)
	err := sdc.CreateService(context.TODO(), test.NsName, test.SvcName)
	assert.Equal(t, nsErr, err)
}

func TestServiceDiscoveryClient_GetService_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdApi.EXPECT().ListNamespaces(context.TODO()).Return([]*model.Namespace{{Id: test.NsId, Name: test.NsName}}, nil)
	sdApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Id: test.SvcId, Name: test.SvcName}}, nil)
	sdApi.EXPECT().ListInstances(context.TODO(), test.SvcId).
		Return([]types.InstanceSummary{
			{
				Id: aws.String(test.EndptId1),
				Attributes: map[string]string{
					model.Ipv4Attr: test.EndptIp1,
					model.PortAttr: test.EndptPortStr1,
				},
			},
			{
				Id: aws.String(test.EndptId2),
				Attributes: map[string]string{
					model.Ipv4Attr: test.EndptIp2,
					model.PortAttr: test.EndptPortStr2,
				},
			},
		}, nil)
	sdc := getTestSdClient(t, sdApi)

	svc, err := sdc.GetService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err)
	assert.Equal(t, test.GetTestService(), svc)
}

func TestServiceDiscoveryClient_GetService_CachedValues(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdc := getTestSdClient(t, sdApi)
	sdc.namespaceCache.Add(test.NsName, *test.GetTestHttpNamespace(), time.Minute)
	sdc.serviceIdCache.Add(fmt.Sprintf("%s/%s", test.NsName, test.SvcName), test.SvcId, time.Minute)
	sdc.endpointCache.Add(test.SvcId, []*model.Endpoint{test.GetTestEndpoint(), test.GetTestEndpoint2()}, time.Minute)

	svc, err := sdc.GetService(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err)
	assert.Equal(t, test.GetTestService(), svc)
}

func TestServiceDiscoveryClient_RegisterEndpoints(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)
	sdc := getTestSdClient(t, sdApi)
	sdc.serviceIdCache.Add(fmt.Sprintf("%s/%s", test.NsName, test.SvcName), test.SvcId, time.Minute)
	sdc.endpointCache.Add(test.SvcId, model.Endpoint{}, time.Minute)

	attrs1 := map[string]string{"AWS_INSTANCE_IPV4": test.EndptIp1, "AWS_INSTANCE_PORT": test.EndptPortStr1}
	attrs2 := map[string]string{"AWS_INSTANCE_IPV4": test.EndptIp2, "AWS_INSTANCE_PORT": test.EndptPortStr2}

	sdApi.EXPECT().RegisterInstance(context.TODO(), test.SvcId, test.EndptId1, attrs1).
		Return(test.OpId1, nil)
	sdApi.EXPECT().RegisterInstance(context.TODO(), test.SvcId, test.EndptId2, attrs2).
		Return(test.OpId2, nil)
	sdApi.EXPECT().ListOperations(context.TODO(), gomock.Any()).
		Return(map[string]types.OperationStatus{
			test.OpId1: types.OperationStatusSuccess,
			test.OpId2: types.OperationStatusSuccess}, nil)

	err := sdc.RegisterEndpoints(context.TODO(), test.NsName, test.SvcName,
		[]*model.Endpoint{
			{
				Id:   test.EndptId1,
				IP:   test.EndptIp1,
				Port: test.EndptPort1,
			},
			{
				Id:   test.EndptId2,
				IP:   test.EndptIp2,
				Port: test.EndptPort2,
			},
		})

	assert.Nil(t, err)
	_, entryCached := sdc.endpointCache.Get(test.SvcId)
	assert.False(t, entryCached, "Cache entry evicted after register")
}

func TestServiceDiscoveryClient_DeleteEndpoints(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	sdc := getTestSdClient(t, sdApi)
	sdc.serviceIdCache.Add(fmt.Sprintf("%s/%s", test.NsName, test.SvcName), test.SvcId, time.Minute)
	sdc.endpointCache.Add(test.SvcId, model.Endpoint{}, time.Minute)

	sdApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1).Return(test.OpId1, nil)
	sdApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId2).Return(test.OpId2, nil)
	sdApi.EXPECT().ListOperations(context.TODO(), gomock.Any()).
		Return(map[string]types.OperationStatus{
			test.OpId1: types.OperationStatusSuccess,
			test.OpId2: types.OperationStatusSuccess}, nil)

	err := sdc.DeleteEndpoints(context.TODO(), test.NsName, test.SvcName,
		[]*model.Endpoint{{Id: test.EndptId1}, {Id: test.EndptId2}})

	assert.Nil(t, err)
	_, entryCached := sdc.endpointCache.Get(test.SvcId)
	assert.False(t, entryCached, "Cache entry evicted after de-register")
}

func TestServiceDiscoveryClient_getNamespace_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	sdc := getTestSdClient(t, sdApi)
	sdc.namespaceCache.Add(test.NsName, *test.GetTestHttpNamespace(), time.Minute)

	namespace, _ := sdc.getNamespace(context.TODO(), test.NsName)
	assert.Equal(t, test.GetTestHttpNamespace(), namespace, "Namespace found in the cache")
}

func TestServiceDiscoveryClient_getNamespace_GetEmptyNamespace(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	sdc := getTestSdClient(t, sdApi)
	sdc.namespaceCache.Add(test.NsName, nil, time.Minute)

	namespace, err := sdc.getNamespace(context.TODO(), test.NsName)
	assert.Nil(t, namespace, "Namespace not found in the cache")
	assert.Nil(t, err, "No errors with empty namespace")
}

func TestServiceDiscoveryClient_getCachedNamespace_ErrorCasting(t *testing.T) {
	sdc := getTestSdClient(t, nil)
	sdc.namespaceCache.Add(test.NsName, struct{ dummy string }{"dummy"}, time.Minute)

	namespace, exists, err := sdc.getCachedNamespace(test.NsName)
	assert.True(t, exists, "Cache exists")
	assert.Nil(t, namespace, "No corresponding cached value found")
	assert.Equal(t, fmt.Sprintf("failed to cast the cached value for the namespace %s", test.NsName), fmt.Sprint(err), "Got the error for improper casting")
}

func getTestSdClient(t *testing.T, sdApi ServiceDiscoveryApi) serviceDiscoveryClient {
	return serviceDiscoveryClient{
		log:            testing2.TestLogger{T: t},
		sdApi:          sdApi,
		namespaceCache: cache.NewLRUExpireCache(1024),
		serviceIdCache: cache.NewLRUExpireCache(1024),
		endpointCache:  cache.NewLRUExpireCache(1024),
	}
}
