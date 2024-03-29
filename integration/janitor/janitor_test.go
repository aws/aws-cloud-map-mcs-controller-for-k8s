package janitor

import (
	"context"
	"testing"

	janitorMock "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/integration/janitor"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testJanitor struct {
	janitor *cloudMapJanitor
	mockApi *janitorMock.MockServiceDiscoveryJanitorApi
	failed  *bool
	close   func()
}

func TestNewDefaultJanitor(t *testing.T) {
	assert.NotNil(t, NewDefaultJanitor(test.ClusterId1, test.ClusterSet))
}

func TestCleanupHappyCase(t *testing.T) {
	tj := getTestJanitor(t)
	defer tj.close()

	tj.mockApi.EXPECT().GetNamespaceMap(context.TODO()).
		Return(map[string]*model.Namespace{test.HttpNsName: test.GetTestHttpNamespace()}, nil)
	tj.mockApi.EXPECT().GetServiceIdMap(context.TODO(), test.HttpNsId).
		Return(map[string]string{test.SvcName: test.SvcId}, nil)
	tj.mockApi.EXPECT().DiscoverInstances(context.TODO(), test.HttpNsName, test.SvcName, map[string]string{
		model.ClusterSetIdAttr: test.ClusterSet,
	}).
		Return([]types.HttpInstanceSummary{{InstanceId: aws.String(test.EndptId1)}}, nil)

	tj.mockApi.EXPECT().DeregisterInstance(context.TODO(), test.SvcId, test.EndptId1).
		Return(test.OpId1, nil)
	tj.mockApi.EXPECT().GetOperation(context.TODO(), test.OpId1).
		Return(&types.Operation{Status: types.OperationStatusSuccess}, nil)
	tj.mockApi.EXPECT().DeleteService(context.TODO(), test.SvcId).
		Return(nil)
	tj.mockApi.EXPECT().DeleteNamespace(context.TODO(), test.HttpNsId).
		Return(test.OpId2, nil)
	tj.mockApi.EXPECT().GetOperation(context.TODO(), test.OpId2).
		Return(&types.Operation{Status: types.OperationStatusSuccess,
			Targets: map[string]string{string(types.OperationTargetTypeNamespace): test.HttpNsId}}, nil)

	tj.janitor.Cleanup(context.TODO(), test.HttpNsName)
	assert.False(t, *tj.failed)
}

func TestCleanupNothingToClean(t *testing.T) {
	tj := getTestJanitor(t)
	defer tj.close()

	tj.mockApi.EXPECT().GetNamespaceMap(context.TODO()).
		Return(map[string]*model.Namespace{}, nil)

	tj.janitor.Cleanup(context.TODO(), test.HttpNsName)
	assert.False(t, *tj.failed)
}

func getTestJanitor(t *testing.T) *testJanitor {
	mockController := gomock.NewController(t)
	api := janitorMock.NewMockServiceDiscoveryJanitorApi(mockController)
	failed := false
	return &testJanitor{
		janitor: &cloudMapJanitor{
			clusterId:    test.ClusterId1,
			clusterSetId: test.ClusterSet,
			sdApi:        api,
			fail:         func() { failed = true },
		},
		mockApi: api,
		failed:  &failed,
		close:   func() { mockController.Finish() },
	}
}
