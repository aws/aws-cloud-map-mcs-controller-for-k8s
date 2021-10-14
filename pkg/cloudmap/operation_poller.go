package cloudmap

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"time"
)

const (
	// Interval between each getOperation call.
	defaultOperationPollInterval = 3 * time.Second

	//
	defaultOperationPollTimeout = 5 * time.Minute
)

// OperationPoller polls a list operations for a terminal status.
type OperationPoller interface {
	// Poll monitors operations until they reach terminal state.
	Poll(ctx context.Context) error
}

type operationPoller struct {
	log   logr.Logger
	sdApi ServiceDiscoveryApi
	opIds []string

	svcId  string
	opType types.OperationType
	start  int
}

func newOperationPoller(sdApi ServiceDiscoveryApi, svcId string, opIds []string, startTime int) operationPoller {
	return operationPoller{
		log:   ctrl.Log.WithName("cloudmap"),
		sdApi: sdApi,

		opIds: opIds,
		svcId: svcId,
		start: startTime,
	}
}

// NewRegisterInstancePoller creates a new operation poller for register instance operations.
func NewRegisterInstancePoller(sdApi ServiceDiscoveryApi, serviceId string, opIds []string, startTime int) OperationPoller {
	poller := newOperationPoller(sdApi, serviceId, opIds, startTime)
	poller.opType = types.OperationTypeRegisterInstance
	return &poller
}

// NewDeregisterInstancePoller creates a new operation poller for de-register instance operations.
func NewDeregisterInstancePoller(sdApi ServiceDiscoveryApi, serviceId string, opIds []string, startTime int) OperationPoller {
	poller := newOperationPoller(sdApi, serviceId, opIds, startTime)
	poller.opType = types.OperationTypeDeregisterInstance
	return &poller
}

func (opPoller *operationPoller) Poll(ctx context.Context) error {
	if len(opPoller.opIds) == 0 {
		opPoller.log.Info("no operations to poll")
		return nil
	}

	return wait.Poll(defaultOperationPollInterval, defaultOperationPollTimeout, func() (done bool, err error) {
		opPoller.log.Info("polling operations", "operations", opPoller.opIds)

		sdOps, err := opPoller.sdApi.ListOperations(ctx, opPoller.buildFilters())

		if err != nil {
			return true, err
		}

		failedOps := make([]string, 0)

		for _, pollOp := range opPoller.opIds {
			status, hasVal := sdOps[pollOp]
			if !hasVal {
				// polled operation not terminal
				return false, nil
			}

			if status == types.OperationStatusFail {
				failedOps = append(failedOps, pollOp)
			}
		}

		if len(failedOps) != 0 {
			for _, failedOp := range failedOps {
				opPoller.log.Info("Operation failed", "failedOp", failedOp, "reason", opPoller.getFailedOpReason(ctx, failedOp))
			}
			return true, fmt.Errorf("operation failure")
		}

		opPoller.log.Info("operations completed successfully")
		return true, nil
	})
}

func (opPoller *operationPoller) buildFilters() []types.OperationFilter {
	svcFilter := types.OperationFilter{
		Name:   types.OperationFilterNameServiceId,
		Values: []string{opPoller.svcId},
	}
	statusFilter := types.OperationFilter{
		Name:      types.OperationFilterNameStatus,
		Condition: types.FilterConditionIn,

		Values: []string{
			string(types.OperationStatusFail),
			string(types.OperationStatusSuccess)},
	}
	typeFilter := types.OperationFilter{
		Name:   types.OperationFilterNameType,
		Values: []string{string(opPoller.opType)},
	}

	timeFilter := types.OperationFilter{
		Name:      types.OperationFilterNameUpdateDate,
		Condition: types.FilterConditionBetween,
		Values: []string{
			strconv.Itoa(opPoller.start),
			// Add one minute to end range in case op updates while list request is in flight
			strconv.Itoa(Now() + 60000),
		},
	}

	return []types.OperationFilter{svcFilter, statusFilter, typeFilter, timeFilter}
}

// getFailedOpReason returns operation error message, which is not available in ListOperations response
func (opPoller *operationPoller) getFailedOpReason(ctx context.Context, opId string) string {
	msg, err := opPoller.sdApi.GetOperationErrorMessage(ctx, opId)

	if err != nil {
		return "failed to retrieve operation failure reason"
	}

	return msg
}

// Now returns current time with milliseconds, as used by operation UPDATE_DATE field
func Now() int {
	return int(time.Now().UnixNano() / 1000000)
}
