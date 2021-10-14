package cloudmap

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OperationCollector collects a list operations with thread safety.
type OperationCollector interface {
	// Add an operation to poll with thread safety, to be called from within a go routine.
	Add(endpointId string, operationId string, operationError error)

	// Collect waits for all operations to be added and returns a list of successfully created operation IDs.
	Collect() []string

	// IsAllOperationsCreated returns true if all operations were created successfully.
	IsAllOperationsCreated() bool
}

type opCollector struct {
	log              logr.Logger
	opChan           chan opResult
	opCount          int
	createOpsSuccess bool
}

type opResult struct {
	instId string
	opId   string
	err    error
}

func NewOperationCollector(opCount int) OperationCollector {
	return &opCollector{
		log:              ctrl.Log.WithName("cloudmap"),
		opChan:           make(chan opResult),
		opCount:          opCount,
		createOpsSuccess: true,
	}
}

func (opColl *opCollector) Add(endptId string, opId string, opErr error) {
	opColl.opChan <- opResult{endptId, opId, opErr}
}

func (opColl *opCollector) Collect() []string {
	opIds := make([]string, 0)

	for i := 0; i < opColl.opCount; i++ {
		op := <-opColl.opChan

		if op.err != nil {
			opColl.log.Info("could not create operation", "error", op.err)
			opColl.createOpsSuccess = false
			continue
		}

		opIds = append(opIds, op.opId)
	}

	return opIds
}

func (opColl *opCollector) IsAllOperationsCreated() bool {
	return opColl.createOpsSuccess
}
