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
	// interval between each getOperation call
	defaultOperationPollInterval = 3 * time.Second
)

type OperationPoller interface {
	// AddOperation adds an operation to poll
	AddOperation(endpointId string, operationId string, operationError error)

	// PollOperations monitors operations until they reach terminal state
	PollOperations(ctx context.Context) error

	// IsAllOperationsCreated returns true if all operations were created successfully
	IsAllOperationsCreated() bool
}

type operationPoller struct {
	log              logr.Logger
	sdApi            ServiceDiscoveryApi
	opChan           chan opResult
	opIds            []string
	createOpsSuccess bool

	svcId   string
	opType  types.OperationType
	opCount int
	start   int64
}

type opResult struct {
	instId string
	opId   string
	err    error
}

func newOperationPoller(sdApi ServiceDiscoveryApi, svcId string, opType types.OperationType, opCount int) OperationPoller {
	return &operationPoller{
		log:              ctrl.Log.WithName("cloudmap"),
		sdApi:            sdApi,
		opChan:           make(chan opResult),
		opIds:            make([]string, opCount),
		createOpsSuccess: true,

		svcId:   svcId,
		opType:  opType,
		opCount: opCount,
		start:   now(),
	}
}

func NewRegisterInstancePoller(sdApi ServiceDiscoveryApi, serviceId string, opCount int) OperationPoller {
	return newOperationPoller(sdApi, serviceId, types.OperationTypeRegisterInstance, opCount)
}

func NewDeregisterInstancePoller(sdApi ServiceDiscoveryApi, serviceId string, opCount int) OperationPoller {
	return newOperationPoller(sdApi, serviceId, types.OperationTypeDeregisterInstance, opCount)
}

func (opPoller *operationPoller) AddOperation(endptId string, opId string, opErr error) {
	opPoller.opChan <- opResult{endptId, opId, opErr}
}

func (opPoller *operationPoller) IsAllOperationsCreated() bool {
	return opPoller.createOpsSuccess
}

func (opPoller *operationPoller) PollOperations(ctx context.Context) error {
	if len(opPoller.opIds) == 0 {
		opPoller.log.Info("no operations to poll")
		return nil
	}

	opPoller.collectOperations()
	return opPoller.pollOperations(ctx)
}

func (opPoller *operationPoller) collectOperations() {
	opPoller.createOpsSuccess = true

	for i := 0; i < opPoller.opCount; i++ {
		op := <-opPoller.opChan

		if op.err != nil {
			opPoller.log.Info("could not create operation", "error", op.err)
			opPoller.createOpsSuccess = false
			continue
		}

		opPoller.opIds[i] = op.opId
	}
}

func (opPoller *operationPoller) pollOperations(ctx context.Context) error {
	if len(opPoller.opIds) == 0 {
		opPoller.log.Info("no operations to poll")
		return nil
	}

	return wait.PollUntil(defaultOperationPollInterval, func() (done bool, err error) {
		opPoller.log.Info("polling operations", "operations", opPoller.opIds)

		sdOps, err := opPoller.sdApi.ListOperations(ctx, opPoller.buildFilters())

		if err != nil {
			return true, nil
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
	}, ctx.Done())
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
			strconv.Itoa(int(opPoller.start)),
			// Add one minute to end range in case op updates while list request is in flight
			strconv.Itoa(int(now() + 60000)),
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

// now returns current time with milliseconds, as used by operation UPDATE_DATE field
func now() int64 {
	return time.Now().UnixNano() / 1000000
}
