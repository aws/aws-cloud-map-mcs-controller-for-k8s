package cloudmap

import (
	"context"
	"errors"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"time"
)

const (
	// Interval between each getOperation call.
	defaultOperationPollInterval = 3 * time.Second

	// Time until we stop polling the operation
	defaultOperationPollTimeout = 5 * time.Minute

	operationPollTimoutErrorMessage = "timed out while polling operations"
)

// OperationPoller polls a list operations for a terminal status.
type OperationPoller interface {
	// Poll monitors operations until they reach terminal state.
	Poll(ctx context.Context) error
}

type operationPoller struct {
	log     common.Logger
	sdApi   ServiceDiscoveryApi
	timeout time.Duration

	opIds  []string
	svcId  string
	opType types.OperationType
	start  int64
}

func newOperationPoller(sdApi ServiceDiscoveryApi, svcId string, opIds []string, startTime int64) operationPoller {
	return operationPoller{
		log:     common.NewLogger(ctrl.Log.WithName("cloudmap")),
		sdApi:   sdApi,
		timeout: defaultOperationPollTimeout,

		opIds: opIds,
		svcId: svcId,
		start: startTime,
	}
}

// NewRegisterInstancePoller creates a new operation poller for register instance operations.
func NewRegisterInstancePoller(sdApi ServiceDiscoveryApi, serviceId string, opIds []string, startTime int64) OperationPoller {
	poller := newOperationPoller(sdApi, serviceId, opIds, startTime)
	poller.opType = types.OperationTypeRegisterInstance
	return &poller
}

// NewDeregisterInstancePoller creates a new operation poller for de-register instance operations.
func NewDeregisterInstancePoller(sdApi ServiceDiscoveryApi, serviceId string, opIds []string, startTime int64) OperationPoller {
	poller := newOperationPoller(sdApi, serviceId, opIds, startTime)
	poller.opType = types.OperationTypeDeregisterInstance
	return &poller
}

func (opPoller *operationPoller) Poll(ctx context.Context) (err error) {
	if len(opPoller.opIds) == 0 {
		opPoller.log.Info("no operations to poll")
		return nil
	}

	err = wait.Poll(defaultOperationPollInterval, opPoller.timeout, func() (done bool, err error) {
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
				opPoller.log.Info("operation failed", "failedOp", failedOp, "reason", opPoller.getFailedOpReason(ctx, failedOp))
			}
			return true, errors.New("operation failure")
		}

		opPoller.log.Info("operations completed successfully")
		return true, nil
	})

	if err == wait.ErrWaitTimeout {
		return errors.New(operationPollTimoutErrorMessage)
	}

	return err
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
			Itoa(opPoller.start),
			// Add one minute to end range in case op updates while list request is in flight
			Itoa(Now() + 60000),
		},
	}

	return []types.OperationFilter{svcFilter, statusFilter, typeFilter, timeFilter}
}

// getFailedOpReason returns operation error message, which is not available in ListOperations response
func (opPoller *operationPoller) getFailedOpReason(ctx context.Context, opId string) string {
	op, err := opPoller.sdApi.GetOperation(ctx, opId)

	if err != nil {
		return "failed to retrieve operation failure reason"
	}

	return aws.ToString(op.ErrorMessage)
}
func Itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Now returns current time with milliseconds, as used by operation filter UPDATE_DATE field
func Now() int64 {
	return time.Now().UnixNano() / 1000000
}
