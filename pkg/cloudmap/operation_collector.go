package cloudmap

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
)

// OperationCollector collects a list of operation IDs asynchronously with thread safety.
type OperationCollector interface {
	// Add calls an operation provider function to asynchronously collect operations to poll.
	Add(operationProvider func() (operationId string, err error))

	// Collect waits for all create operation results to be provided and returns a list of the successfully created operation IDs.
	Collect() []string

	// GetStartTime returns the start time range to poll the collected operations.
	GetStartTime() int64

	// IsAllOperationsCreated returns true if all operations were created successfully.
	IsAllOperationsCreated() bool
}

type opCollector struct {
	log              logr.Logger
	opChan           chan opResult
	wg               sync.WaitGroup
	startTime        int64
	createOpsSuccess bool
}

type opResult struct {
	opId string
	err  error
}

func NewOperationCollector() OperationCollector {
	return &opCollector{
		log:              ctrl.Log.WithName("cloudmap"),
		opChan:           make(chan opResult),
		startTime:        Now(),
		createOpsSuccess: true,
	}
}

func (opColl *opCollector) Add(opProvider func() (opId string, err error)) {
	opColl.wg.Add(1)
	go func() {
		defer opColl.wg.Done()

		opId, opErr := opProvider()
		opColl.opChan <- opResult{opId, opErr}
	}()
}

func (opColl *opCollector) Collect() []string {
	opIds := make([]string, 0)

	// Run wait in separate go routine to unblock reading from the channel.
	go func() {
		opColl.wg.Wait()
		close(opColl.opChan)
	}()

	for op := range opColl.opChan {
		if op.err != nil {
			opColl.log.Info("could not create operation", "error", op.err)
			opColl.createOpsSuccess = false
			continue
		}

		opIds = append(opIds, op.opId)
	}

	return opIds
}

func (opColl *opCollector) GetStartTime() int64 {
	return opColl.startTime
}

func (opColl *opCollector) IsAllOperationsCreated() bool {
	return opColl.createOpsSuccess
}
