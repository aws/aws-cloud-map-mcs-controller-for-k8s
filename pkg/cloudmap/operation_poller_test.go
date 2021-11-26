package cloudmap

import (
	"context"
	"errors"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	testing2 "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func TestOperationPoller_HappyCases(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	pollerTypes := []struct {
		constructor    func() OperationPoller
		expectedOpType types.OperationType
	}{
		{
			constructor: func() OperationPoller {
				return NewRegisterInstancePoller(sdApi, test.SvcId, []string{test.OpId1, test.OpId2}, test.OpStart)
			},
			expectedOpType: types.OperationTypeRegisterInstance,
		},
		{
			constructor: func() OperationPoller {
				return NewDeregisterInstancePoller(sdApi, test.SvcId, []string{test.OpId1, test.OpId2}, test.OpStart)
			},
			expectedOpType: types.OperationTypeDeregisterInstance,
		},
	}

	for _, pollerType := range pollerTypes {
		p := pollerType.constructor()

		var firstEnd int

		sdApi.EXPECT().
			ListOperations(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, filters []types.OperationFilter) (map[string]types.OperationStatus, error) {
				assert.Contains(t, filters,
					types.OperationFilter{
						Name:   types.OperationFilterNameServiceId,
						Values: []string{test.SvcId},
					})
				assert.Contains(t, filters,
					types.OperationFilter{
						Name:      types.OperationFilterNameStatus,
						Condition: types.FilterConditionIn,

						Values: []string{
							string(types.OperationStatusFail),
							string(types.OperationStatusSuccess)},
					})
				assert.Contains(t, filters,
					types.OperationFilter{
						Name:   types.OperationFilterNameType,
						Values: []string{string(pollerType.expectedOpType)},
					})

				timeFilter := findUpdateDateFilter(t, filters)
				assert.NotNil(t, timeFilter)
				assert.Equal(t, types.FilterConditionBetween, timeFilter.Condition)
				assert.Equal(t, 2, len(timeFilter.Values))

				filterStart, _ := strconv.Atoi(timeFilter.Values[0])
				assert.Equal(t, test.OpStart, filterStart)

				firstEnd, _ = strconv.Atoi(timeFilter.Values[1])

				return map[string]types.OperationStatus{}, nil
			})

		sdApi.EXPECT().
			ListOperations(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, filters []types.OperationFilter) (map[string]types.OperationStatus, error) {
				timeFilter := findUpdateDateFilter(t, filters)
				secondEnd, _ := strconv.Atoi(timeFilter.Values[1])
				assert.Greater(t, secondEnd, firstEnd,
					"Filter time frame for operations must increase between invocations of ListOperations")

				return map[string]types.OperationStatus{
					test.OpId1: types.OperationStatusSuccess,
					test.OpId2: types.OperationStatusSuccess,
				}, nil
			})

		err := p.Poll(context.TODO())

		assert.Nil(t, err)
	}
}

func TestOperationPoller_PollEmpty(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	p := NewRegisterInstancePoller(sdApi, test.SvcId, []string{}, test.OpStart)
	err := p.Poll(context.TODO())
	assert.Nil(t, err)
}

func TestOperationPoller_PollFailure(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	p := NewRegisterInstancePoller(sdApi, test.SvcId, []string{test.OpId1, test.OpId2}, test.OpStart)

	pollErr := errors.New("error polling operations")

	sdApi.EXPECT().
		ListOperations(gomock.Any(), gomock.Any()).
		Return(map[string]types.OperationStatus{}, pollErr)

	err := p.Poll(context.TODO())
	assert.Equal(t, pollErr, err)
}

func TestOperationPoller_PollOpFailure(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	p := NewRegisterInstancePoller(sdApi, test.SvcId, []string{test.OpId1, test.OpId2}, test.OpStart)

	sdApi.EXPECT().
		ListOperations(gomock.Any(), gomock.Any()).
		Return(
			map[string]types.OperationStatus{
				test.OpId1: types.OperationStatusSuccess,
				test.OpId2: types.OperationStatusFail,
			}, nil)

	opErr := "operation failure message"

	sdApi.EXPECT().
		GetOperation(gomock.Any(), test.OpId2).
		Return(&types.Operation{ErrorMessage: &opErr}, nil)

	err := p.Poll(context.TODO())
	assert.Equal(t, "operation failure", err.Error())
}

func TestOperationPoller_PollOpFailureAndMessageFailure(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	p := NewRegisterInstancePoller(sdApi, test.SvcId, []string{test.OpId1, test.OpId2}, test.OpStart)

	sdApi.EXPECT().
		ListOperations(gomock.Any(), gomock.Any()).
		Return(
			map[string]types.OperationStatus{
				test.OpId1: types.OperationStatusFail,
				test.OpId2: types.OperationStatusSuccess,
			}, nil)

	sdApi.EXPECT().
		GetOperation(gomock.Any(), test.OpId1).
		Return(nil, errors.New("failed to retrieve operation failure reason"))

	err := p.Poll(context.TODO())
	assert.Equal(t, "operation failure", err.Error())
}

func TestOperationPoller_PollTimeout(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmap.NewMockServiceDiscoveryApi(mockController)

	p := operationPoller{
		log:     common.NewLogger(testing2.TestLogger{T: t}),
		sdApi:   sdApi,
		timeout: 2 * time.Millisecond,
		opIds:   []string{test.OpId1, test.OpId2},
	}

	sdApi.EXPECT().
		ListOperations(gomock.Any(), gomock.Any()).
		Return(
			map[string]types.OperationStatus{}, nil)

	err := p.Poll(context.TODO())
	assert.Equal(t, operationPollTimoutErrorMessage, err.Error())
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "7", Itoa(7))
}

func TestNow(t *testing.T) {
	now1 := Now()
	time.Sleep(time.Millisecond * 5)
	now2 := Now()
	assert.Greater(t, now2, now1)
}

func findUpdateDateFilter(t *testing.T, filters []types.OperationFilter) *types.OperationFilter {
	for _, filter := range filters {
		if filter.Name == types.OperationFilterNameUpdateDate {
			return &filter
		}
	}

	t.Errorf("Missing update date filter")
	return nil
}
