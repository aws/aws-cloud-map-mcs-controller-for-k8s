package janitor

import (
	"context"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/integration/janitor"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testJanitor struct {
	janitor *cloudMapJanitor
	mockApi *janitor.MockServiceDiscoveryJanitorApi
	failed  *bool
	close   func()
}

func TestNewDefaultJanitor(t *testing.T) {
	assert.NotNil(t, NewDefaultJanitor())
}

func TestCleanupHappyCase(t *testing.T) {
	tj := getTestJanitor(t)
	defer tj.close()

	tj.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{{Id: test.NsId, Name: test.NsName}}, nil)
	tj.mockApi.EXPECT().ListServices(context.TODO(), test.NsId).
		Return([]*model.Resource{{Id: test.SvcId, Name: test.SvcName}}, nil)
	tj.mockApi.EXPECT().ListInstances(context.TODO(), test.SvcId).
		Return([]types.InstanceSummary{{Id: aws.String(test.EndptId1)}}, nil)

	tj.mockApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1).
		Return(test.OpId1, nil)
	tj.mockApi.EXPECT().ListOperations(context.TODO(), gomock.Any()).
		Return(map[string]types.OperationStatus{test.OpId1: types.OperationStatusSuccess}, nil)
	tj.mockApi.EXPECT().DeleteService(context.TODO(), test.SvcId).
		Return(nil)
	tj.mockApi.EXPECT().DeleteNamespace(context.TODO(), test.NsId).
		Return(test.OpId2, nil)
	tj.mockApi.EXPECT().PollNamespaceOperation(context.TODO(), test.OpId2).
		Return(test.NsId, nil)

	tj.janitor.Cleanup(context.TODO(), test.NsName)
	assert.False(t, *tj.failed)
}

func TestCleanupNothingToClean(t *testing.T) {
	tj := getTestJanitor(t)
	defer tj.close()

	tj.mockApi.EXPECT().ListNamespaces(context.TODO()).
		Return([]*model.Namespace{}, nil)

	tj.janitor.Cleanup(context.TODO(), test.NsName)
	assert.False(t, *tj.failed)
}

func getTestJanitor(t *testing.T) *testJanitor {
	mockController := gomock.NewController(t)
	api := janitor.NewMockServiceDiscoveryJanitorApi(mockController)
	failed := false
	return &testJanitor{
		janitor: &cloudMapJanitor{
			sdApi: api,
			fail:  func() { failed = true },
		},
		mockApi: api,
		failed:  &failed,
		close:   func() { mockController.Finish() },
	}
}
