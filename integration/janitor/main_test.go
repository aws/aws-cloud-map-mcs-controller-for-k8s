package main

import (
	"context"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testJanitor struct {
	janitor *janitor
	mockSdk *cloudmap.MockAwsFacade
	mockApi *cloudmap.MockServiceDiscoveryApi
	failed  *bool
	close   func()
}

func TestNewDefaultJanitor(t *testing.T) {
	assert.NotNil(t, newDefaultJanitor())
}

func TestCleanupHappyCase(t *testing.T) {
	tj := getTestJanitor(t)
	defer tj.close()

	tj.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{{Id: test.NsId, Name: test.NsName}}, nil)
	tj.mockApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Id: test.SvcId, Name: test.SvcName}}, nil)
	tj.mockSdk.EXPECT().ListInstances(context.TODO(), gomock.Any()).
		Return(&sd.ListInstancesOutput{
			Instances: []types.InstanceSummary{{Id: aws.String(test.EndptId1)}},
		}, nil)
	tj.mockApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1).
		Return(test.OpId1, nil)
	tj.mockApi.EXPECT().ListOperations(context.TODO(), gomock.Any()).
		Return(map[string]types.OperationStatus{test.OpId1: types.OperationStatusSuccess}, nil)
	tj.mockSdk.EXPECT().DeleteService(context.TODO(), &sd.DeleteServiceInput{Id: aws.String(test.SvcId)}).
		Return(nil, nil)
	tj.mockSdk.EXPECT().DeleteNamespace(context.TODO(), &sd.DeleteNamespaceInput{Id: aws.String(test.NsId)}).
		Return(&sd.DeleteNamespaceOutput{OperationId: aws.String(test.OpId2)}, nil)
	tj.mockApi.EXPECT().PollNamespaceOperation(context.TODO(), test.OpId2)

	tj.janitor.Cleanup(context.TODO(), test.NsName)
	assert.False(t, *tj.failed)
}

func getTestJanitor(t *testing.T) *testJanitor {
	mockController := gomock.NewController(t)
	sdk := cloudmap.NewMockAwsFacade(mockController)
	api := cloudmap.NewMockServiceDiscoveryApi(mockController)
	failed := false
	return &testJanitor{
		janitor: &janitor{
			awsSdk: sdk,
			sdApi:  api,
			fail:   func() { failed = true },
		},
		mockSdk: sdk,
		mockApi: api,
		failed:  &failed,
		close:   func() { mockController.Finish() },
	}
}
