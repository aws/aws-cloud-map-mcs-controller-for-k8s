package cloudmap

import (
	"context"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	testingLogger "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
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
	assert.Equal(t, test.GetTestDnsNamespace(), namespaces[0], "")
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
	assert.Equal(t, "dummy error", fmt.Sprint(err), "Got error")
}

func getServiceDiscoveryApi(t *testing.T, awsFacade *cloudmap.MockAwsFacade) serviceDiscoveryApi {
	return serviceDiscoveryApi{
		log:       testingLogger.TestLogger{T: t},
		awsFacade: awsFacade,
	}
}
