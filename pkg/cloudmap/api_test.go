package cloudmap

import (
	"context"
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

func TestListNamespaces_HappyCase(t *testing.T) {
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

func TestListNamespaces_SkipPublicDNSNotSupported(t *testing.T) {
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

func getServiceDiscoveryApi(t *testing.T, awsFacade *cloudmap.MockAwsFacade) serviceDiscoveryApi {
	return serviceDiscoveryApi{
		log:       testingLogger.TestLogger{T: t},
		awsFacade: awsFacade,
	}
}
