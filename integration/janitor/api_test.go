package janitor

import (
	"context"
	"testing"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/integration/janitor"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewServiceDiscoveryJanitorApiFromConfig(t *testing.T) {
	assert.NotNil(t, NewServiceDiscoveryJanitorApiFromConfig(&aws.Config{}))
}

func TestServiceDiscoveryJanitorApi_DeleteNamespace_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mocksdk := janitor.NewMockSdkJanitorFacade(mockController)
	jApi := getJanitorApi(mocksdk)

	mocksdk.EXPECT().DeleteNamespace(context.TODO(), &sd.DeleteNamespaceInput{Id: aws.String(test.NsId)}).
		Return(&sd.DeleteNamespaceOutput{OperationId: aws.String(test.OpId1)}, nil)

	opId, err := jApi.DeleteNamespace(context.TODO(), test.NsId)
	assert.Nil(t, err, "No error for happy case")
	assert.Equal(t, test.OpId1, opId)
}

func TestServiceDiscoveryJanitorApi_DeleteService_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	mocksdk := janitor.NewMockSdkJanitorFacade(mockController)
	jApi := getJanitorApi(mocksdk)

	mocksdk.EXPECT().DeleteService(context.TODO(), &sd.DeleteServiceInput{Id: aws.String(test.SvcId)}).
		Return(&sd.DeleteServiceOutput{}, nil)

	err := jApi.DeleteService(context.TODO(), test.SvcId)
	assert.Nil(t, err, "No error for happy case")
}

func getJanitorApi(sdk *janitor.MockSdkJanitorFacade) ServiceDiscoveryJanitorApi {
	return &serviceDiscoveryJanitorApi{
		janitorFacade: sdk,
	}
}
