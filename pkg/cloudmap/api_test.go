package cloudmap

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	testingLogger "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewServiceDiscoveryApi(t *testing.T) {
	sdc := NewServiceDiscoveryApiFromConfig(&aws.Config{})
	assert.NotNil(t, sdc)
}

func TestServiceDiscoveryApi_ListNamespaces_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	id, name := test.NsId, test.NsName
	ns := types.NamespaceSummary{
		Name: &name,
		Id:   &id,
		Type: types.NamespaceTypeDnsPrivate,
	}
	awsFacade.EXPECT().ListNamespaces(context.TODO(), &sd.ListNamespacesInput{}).
		Return(&sd.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{ns}}, nil)

	namespaces, _ := sdApi.ListNamespaces(context.TODO())
	assert.True(t, len(namespaces) == 1)
	assert.Equal(t, test.GetTestDnsNamespace(), namespaces[0], "No error for happy case")
}

func TestServiceDiscoveryApi_ListNamespaces_SkipPublicDNSNotSupported(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	id, name := test.NsId, test.NsName
	ns := types.NamespaceSummary{
		Name: &name,
		Id:   &id,
		Type: types.NamespaceTypeDnsPublic,
	}
	awsFacade.EXPECT().ListNamespaces(context.TODO(), &sd.ListNamespacesInput{}).
		Return(&sd.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{ns}}, nil)

	namespaces, _ := sdApi.ListNamespaces(context.TODO())
	assert.True(t, len(namespaces) == 0, "Successfully skipped DNS_PUBLIC from the output")
}

func TestServiceDiscoveryApi_ListServices_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	filter := types.ServiceFilter{
		Name:   types.ServiceFilterNameNamespaceId,
		Values: []string{test.NsId},
	}

	awsFacade.EXPECT().ListServices(context.TODO(), &sd.ListServicesInput{Filters: []types.ServiceFilter{filter}}).
		Return(&sd.ListServicesOutput{Services: []types.ServiceSummary{
			{Id: aws.String(test.SvcId), Name: aws.String(test.SvcName)},
		}}, nil)

	svcs, err := sdApi.ListServices(context.TODO(), test.NsId)
	assert.Nil(t, err, "No error for happy case")
	assert.True(t, len(svcs) == 1)
	assert.Equal(t, svcs[0], &model.Resource{Id: test.SvcId, Name: test.SvcName})
}

func TestServiceDiscoveryApi_DiscoverInstances_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	awsFacade.EXPECT().DiscoverInstances(context.TODO(),
		&sd.DiscoverInstancesInput{
			NamespaceName: aws.String(test.NsName),
			ServiceName:   aws.String(test.SvcName),
			HealthStatus:  types.HealthStatusFilterAll,
			MaxResults:    aws.Int32(1000),
		}).
		Return(&sd.DiscoverInstancesOutput{
			Instances: []types.HttpInstanceSummary{
				{InstanceId: aws.String(test.EndptId1)},
				{InstanceId: aws.String(test.EndptId2)},
			},
		}, nil)

	insts, err := sdApi.DiscoverInstances(context.TODO(), test.NsName, test.SvcName)
	assert.Nil(t, err, "No error for happy case")
	assert.True(t, len(insts) == 2)
	assert.Equal(t, test.EndptId1, *insts[0].InstanceId)
	assert.Equal(t, test.EndptId2, *insts[1].InstanceId)
}

func TestServiceDiscoveryApi_ListOperations_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
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

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
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

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	awsFacade.EXPECT().CreateHttpNamespace(context.TODO(), &sd.CreateHttpNamespaceInput{Name: aws.String(test.NsName)}).
		Return(&sd.CreateHttpNamespaceOutput{OperationId: aws.String(test.OpId1)}, nil)

	opId, err := sdApi.CreateHttpNamespace(context.TODO(), test.NsName)
	assert.Nil(t, err, "No error for happy case")
	assert.Equal(t, test.OpId1, opId)
}

func TestServiceDiscoveryApi_CreateService_CreateForHttpNamespace(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	nsId, svcId, svcName := test.NsId, test.SvcId, test.SvcName
	awsFacade.EXPECT().CreateService(context.TODO(), &sd.CreateServiceInput{
		Name:        &svcName,
		NamespaceId: &nsId,
	}).
		Return(&sd.CreateServiceOutput{
			Service: &types.Service{
				Id: &svcId,
			},
		}, nil)

	retSvcId, _ := sdApi.CreateService(context.TODO(), *test.GetTestHttpNamespace(), svcName)
	assert.Equal(t, svcId, retSvcId, "Successfully created service")
}

func TestServiceDiscoveryApi_CreateService_CreateForDnsNamespace(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	nsId, svcId, svcName := test.NsId, test.SvcId, test.SvcName
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

	retSvcId, _ := sdApi.CreateService(context.TODO(), *test.GetTestDnsNamespace(), svcName)
	assert.Equal(t, svcId, retSvcId, "Successfully created service")
}

func TestServiceDiscoveryApi_CreateService_ThrowError(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	sdApi := getServiceDiscoveryApi(t, awsFacade)

	nsId, svcName := test.NsId, test.SvcName
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

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
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
	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	awsFacade.EXPECT().RegisterInstance(context.TODO(), gomock.Any()).Return(nil, sdkErr)

	sdApi := getServiceDiscoveryApi(t, awsFacade)
	_, err := sdApi.RegisterInstance(context.TODO(), test.SvcId, test.EndptId1, map[string]string{})
	assert.Equal(t, sdkErr, err)
}

func TestServiceDiscoveryApi_DeregisterInstance_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
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
	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	awsFacade.EXPECT().DeregisterInstance(context.TODO(), gomock.Any()).Return(nil, sdkErr)

	sdApi := getServiceDiscoveryApi(t, awsFacade)
	_, err := sdApi.DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1)
	assert.Equal(t, sdkErr, err)
}

func TestServiceDiscoveryApi_PollNamespaceOperation_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	awsFacade := cloudmap.NewMockAwsFacade(mockController)
	awsFacade.EXPECT().GetOperation(context.TODO(), &sd.GetOperationInput{OperationId: aws.String(test.OpId1)}).
		Return(&sd.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusPending}}, nil)

	awsFacade.EXPECT().GetOperation(context.TODO(), &sd.GetOperationInput{OperationId: aws.String(test.OpId1)}).
		Return(&sd.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusSuccess,
			Targets: map[string]string{string(types.OperationTargetTypeNamespace): test.NsId}}}, nil)

	sdApi := getServiceDiscoveryApi(t, awsFacade)

	nsId, err := sdApi.PollNamespaceOperation(context.TODO(), test.OpId1)
	assert.Nil(t, err)
	assert.Equal(t, test.NsId, nsId)
}

func getServiceDiscoveryApi(t *testing.T, awsFacade *cloudmap.MockAwsFacade) ServiceDiscoveryApi {
	return &serviceDiscoveryApi{
		log:       common.NewLoggerWithLogr(testingLogger.TestLogger{T: t}),
		awsFacade: awsFacade,
	}
}
