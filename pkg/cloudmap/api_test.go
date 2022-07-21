package cloudmap

import (
	"context"
	"errors"
	"fmt"
	"testing"

	cloudmapMock "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/go-logr/logr/testr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewServiceDiscoveryApi(t *testing.T) {
	sdc := NewServiceDiscoveryApiFromConfig(&aws.Config{})
	assert.NotNil(t, sdc)
}

func TestServiceDiscoveryApi_GetNamespaceMap_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	id, name := test.DnsNsId, test.DnsNsName
	ns := types.NamespaceSummary{
		Name: &name,
		Id:   &id,
		Type: types.NamespaceTypeDnsPrivate,
	}
	awsFacade.EXPECT().ListNamespaces(context.TODO(), &sd.ListNamespacesInput{}).
		Return(&sd.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{ns}}, nil)

	namespaces, err := sdApi.GetNamespaceMap(context.TODO())
	assert.Nil(t, err, "No error for happy case")
	assert.True(t, len(namespaces) == 1)
	assert.Equal(t, test.GetTestDnsNamespace(), namespaces[test.DnsNsName])
}

func TestServiceDiscoveryApi_GetNamespaceMap_SkipPublicDNSNotSupported(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	id, name := test.DnsNsId, test.DnsNsName
	ns := types.NamespaceSummary{
		Name: &name,
		Id:   &id,
		Type: types.NamespaceTypeDnsPublic,
	}
	awsFacade.EXPECT().ListNamespaces(context.TODO(), &sd.ListNamespacesInput{}).
		Return(&sd.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{ns}}, nil)

	namespaces, err := sdApi.GetNamespaceMap(context.TODO())
	assert.Nil(t, err, "No error for happy case")
	assert.Empty(t, namespaces, "Successfully skipped DNS_PUBLIC from the output")
}

func TestServiceDiscoveryApi_GetServiceIdMap_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	filter := types.ServiceFilter{
		Name:   types.ServiceFilterNameNamespaceId,
		Values: []string{test.HttpNsId},
	}

	awsFacade.EXPECT().ListServices(context.TODO(), &sd.ListServicesInput{Filters: []types.ServiceFilter{filter}}).
		Return(&sd.ListServicesOutput{Services: []types.ServiceSummary{
			{Id: aws.String(test.SvcId), Name: aws.String(test.SvcName)},
		}}, nil)

	svcs, err := sdApi.GetServiceIdMap(context.TODO(), test.HttpNsId)
	assert.Nil(t, err, "No error for happy case")
	assert.True(t, len(svcs) == 1)
	assert.Equal(t, svcs[test.SvcName], test.SvcId)
}

func TestServiceDiscoveryApi_DiscoverInstances_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	awsFacade.EXPECT().DiscoverInstances(context.TODO(),
		&sd.DiscoverInstancesInput{
			NamespaceName: aws.String(test.HttpNsName),
			ServiceName:   aws.String(test.SvcName),
			HealthStatus:  types.HealthStatusFilterAll,
			MaxResults:    aws.Int32(1000),
			QueryParameters: map[string]string{
				model.ClustersetIdAttr: test.ClustersetId,
			},
		}).
		Return(&sd.DiscoverInstancesOutput{
			Instances: []types.HttpInstanceSummary{
				{InstanceId: aws.String(test.EndptId1)},
				{InstanceId: aws.String(test.EndptId2)},
			},
		}, nil)

	insts, err := sdApi.DiscoverInstances(context.TODO(), test.HttpNsName, test.SvcName, test.ClustersetId)
	assert.Nil(t, err, "No error for happy case")
	assert.True(t, len(insts) == 2)
	assert.Equal(t, test.EndptId1, *insts[0].InstanceId)
	assert.Equal(t, test.EndptId2, *insts[1].InstanceId)
}

func TestServiceDiscoveryApi_ListOperations_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	filters := make([]types.OperationFilter, 0)
	awsFacade.EXPECT().ListOperations(context.TODO(), &sd.ListOperationsInput{Filters: filters}).
		Return(&sd.ListOperationsOutput{
			Operations: []types.OperationSummary{
				{Id: aws.String(test.OpId1), Status: types.OperationStatusSuccess},
			}}, nil)

	ops, err := sdApi.ListOperations(context.TODO(), filters)
	assert.Nil(t, err, "No error for happy case")
	assert.True(t, len(ops) == 1)
	assert.Equal(t, ops[test.OpId1], types.OperationStatusSuccess)
}

func TestServiceDiscoveryApi_GetOperation_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	expectedOp := &types.Operation{Id: aws.String(test.OpId1), Status: types.OperationStatusPending}
	awsFacade.EXPECT().GetOperation(context.TODO(), &sd.GetOperationInput{OperationId: aws.String(test.OpId1)}).
		Return(&sd.GetOperationOutput{Operation: expectedOp}, nil)

	op, err := sdApi.GetOperation(context.TODO(), test.OpId1)
	assert.Nil(t, err, "No error for happy case")
	assert.Equal(t, expectedOp, op)
}

func TestServiceDiscoveryApi_CreateHttNamespace_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	awsFacade.EXPECT().CreateHttpNamespace(context.TODO(), &sd.CreateHttpNamespaceInput{Name: aws.String(test.HttpNsName)}).
		Return(&sd.CreateHttpNamespaceOutput{OperationId: aws.String(test.OpId1)}, nil)

	opId, err := sdApi.CreateHttpNamespace(context.TODO(), test.HttpNsName)
	assert.Nil(t, err, "No error for happy case")
	assert.Equal(t, test.OpId1, opId)
}

func TestServiceDiscoveryApi_CreateService_CreateForHttpNamespace(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	nsId, svcId, svcName := test.HttpNsId, test.SvcId, test.SvcName
	awsFacade.EXPECT().CreateService(context.TODO(), &sd.CreateServiceInput{
		Name:        &svcName,
		NamespaceId: &nsId,
	}).
		Return(&sd.CreateServiceOutput{
			Service: &types.Service{
				Id: &svcId,
			},
		}, nil)

	retSvcId, err := sdApi.CreateService(context.TODO(), *test.GetTestHttpNamespace(), svcName)
	assert.Nil(t, err)
	assert.Equal(t, svcId, retSvcId, "Successfully created service")
}

func TestServiceDiscoveryApi_CreateService_CreateForDnsNamespace(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	nsId, svcId, svcName := test.DnsNsId, test.SvcId, test.SvcName
	awsFacade.EXPECT().CreateService(context.TODO(), &sd.CreateServiceInput{
		Name:        &svcName,
		NamespaceId: &nsId,
		DnsConfig: &types.DnsConfig{
			DnsRecords: []types.DnsRecord{{
				TTL:  aws.Int64(60),
				Type: "SRV",
			}},
		},
	}).
		Return(&sd.CreateServiceOutput{
			Service: &types.Service{
				Id: &svcId,
			},
		}, nil)

	retSvcId, err := sdApi.CreateService(context.TODO(), *test.GetTestDnsNamespace(), svcName)
	assert.Nil(t, err)
	assert.Equal(t, svcId, retSvcId, "Successfully created service")
}

func TestServiceDiscoveryApi_CreateService_ThrowError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	nsId, svcName := test.HttpNsId, test.SvcName
	awsFacade.EXPECT().CreateService(context.TODO(), &sd.CreateServiceInput{
		Name:        &svcName,
		NamespaceId: &nsId,
	}).
		Return(nil, fmt.Errorf("dummy error"))

	retSvcId, err := sdApi.CreateService(context.TODO(), *test.GetTestHttpNamespace(), svcName)
	assert.Empty(t, retSvcId)
	assert.Equal(t, "dummy error", err.Error(), "Got error")
}

func TestServiceDiscoveryApi_RegisterInstance_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	attrs := map[string]string{"a": "b"}

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	awsFacade.EXPECT().RegisterInstance(context.TODO(),
		&sd.RegisterInstanceInput{
			ServiceId:  aws.String(test.SvcId),
			InstanceId: aws.String(test.EndptId1),
			Attributes: attrs}).
		Return(&sd.RegisterInstanceOutput{OperationId: aws.String(test.OpId1)}, nil)

	sdApi := getServiceDiscoveryApi(t, awsFacade)
	opId, err := sdApi.RegisterInstance(context.TODO(), test.SvcId, test.EndptId1, attrs)
	assert.Nil(t, err)
	assert.Equal(t, test.OpId1, opId)
}

func TestServiceDiscoveryApi_RegisterInstance_Error(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdkErr := errors.New("fail")
	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	awsFacade.EXPECT().RegisterInstance(context.TODO(), gomock.Any()).Return(nil, sdkErr)

	sdApi := getServiceDiscoveryApi(t, awsFacade)
	_, err := sdApi.RegisterInstance(context.TODO(), test.SvcId, test.EndptId1, map[string]string{})
	assert.Equal(t, sdkErr, err)
}

func TestServiceDiscoveryApi_DeregisterInstance_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	awsFacade.EXPECT().DeregisterInstance(context.TODO(),
		&sd.DeregisterInstanceInput{
			ServiceId:  aws.String(test.SvcId),
			InstanceId: aws.String(test.EndptId1)}).
		Return(&sd.DeregisterInstanceOutput{OperationId: aws.String(test.OpId1)}, nil)

	sdApi := getServiceDiscoveryApi(t, awsFacade)
	opId, err := sdApi.DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1)
	assert.Nil(t, err)
	assert.Equal(t, test.OpId1, opId)
}

func TestServiceDiscoveryApi_DeregisterInstance_Error(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdkErr := errors.New("fail")
	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	awsFacade.EXPECT().DeregisterInstance(context.TODO(), gomock.Any()).Return(nil, sdkErr)

	sdApi := getServiceDiscoveryApi(t, awsFacade)
	_, err := sdApi.DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1)
	assert.Equal(t, sdkErr, err)
}

func TestServiceDiscoveryApi_PollNamespaceOperation_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmapMock.NewMockAwsFacade(mockController)
	awsFacade.EXPECT().GetOperation(context.TODO(), &sd.GetOperationInput{OperationId: aws.String(test.OpId1)}).
		Return(&sd.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusPending}}, nil)

	awsFacade.EXPECT().GetOperation(context.TODO(), &sd.GetOperationInput{OperationId: aws.String(test.OpId1)}).
		Return(&sd.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusSuccess,
			Targets: map[string]string{string(types.OperationTargetTypeNamespace): test.HttpNsId}}}, nil)

	sdApi := getServiceDiscoveryApi(t, awsFacade)

	nsId, err := sdApi.PollNamespaceOperation(context.TODO(), test.OpId1)
	assert.Nil(t, err)
	assert.Equal(t, test.HttpNsId, nsId)
}

func getServiceDiscoveryApi(t *testing.T, awsFacade *cloudmapMock.MockAwsFacade) ServiceDiscoveryApi {
	return &serviceDiscoveryApi{
		log:       common.NewLoggerWithLogr(testr.New(t)),
		awsFacade: awsFacade,
	}
}
