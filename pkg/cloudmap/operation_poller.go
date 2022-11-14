package cloudmap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Interval between each getOperation call.
	defaultOperationPollInterval = 2 * time.Second

	// Time until we stop polling the operation
	defaultOperationPollTimeout = 1 * time.Minute

	operationPollTimoutErrorMessage = "timed out while polling operations"
)

// OperationPoller polls a list operations for a terminal status
type OperationPoller interface {
	// Submit operations to async poll
	Submit(ctx context.Context, opProvider func() (opId string, err error))

	// Poll operations for a terminal state
	Poll(ctx context.Context, opId string) (*types.Operation, error)

	// Await waits for all operation results from async poll
	Await() (err error)
}

type operationPoller struct {
	log          common.Logger
	sdApi        ServiceDiscoveryApi
	opChan       chan opResult
	waitGroup    sync.WaitGroup
	pollInterval time.Duration
	pollTimeout  time.Duration
}

type opResult struct {
	opId string
	err  error
}

// NewOperationPoller creates a new operation poller
func NewOperationPoller(sdApi ServiceDiscoveryApi) OperationPoller {
	return NewOperationPollerWithConfig(defaultOperationPollInterval, defaultOperationPollTimeout, sdApi)
}

// NewOperationPollerWithConfig creates a new operation poller
func NewOperationPollerWithConfig(pollInterval, pollTimeout time.Duration, sdApi ServiceDiscoveryApi) OperationPoller {
	return &operationPoller{
		log:          common.NewLogger("cloudmap", "OperationPoller"),
		sdApi:        sdApi,
		opChan:       make(chan opResult),
		pollInterval: pollInterval,
		pollTimeout:  pollTimeout,
	}
}

func (p *operationPoller) Submit(ctx context.Context, opProvider func() (opId string, err error)) {
	p.waitGroup.Add(1)

	// Poll for the operation in a separate go routine
	go func() {
		// Indicate the polling done i.e. decrement the WaitGroup counter when the goroutine returns
		defer p.waitGroup.Done()

		opId, err := opProvider()
		// Poll for the operationId if the provider doesn't throw error
		if err == nil {
			_, err = p.Poll(ctx, opId)
		}

		p.opChan <- opResult{opId: opId, err: err}
	}()
}

func (p *operationPoller) Poll(ctx context.Context, opId string) (op *types.Operation, err error) {
	// poll tries a condition func until it returns true, an error, or the timeout is reached.
	err = wait.Poll(p.pollInterval, p.pollTimeout, func() (done bool, err error) {
		p.log.Info("polling operation", "opId", opId)

		op, err = p.sdApi.GetOperation(ctx, opId)
		if err != nil {
			return true, err
		}

		switch op.Status {
		case types.OperationStatusSuccess:
			return true, nil
		case types.OperationStatusFail:
			return true, fmt.Errorf("operation failed, opId: %s, reason: %s", opId, aws.ToString(op.ErrorMessage))
		default:
			return false, nil
		}
	})
	if err == wait.ErrWaitTimeout {
		err = fmt.Errorf("%s, opId: %s", operationPollTimoutErrorMessage, opId)
	}

	return op, err
}

func (p *operationPoller) Await() (err error) {
	// Run wait in separate go routine to unblock reading from the channel.
	go func() {
		// Block till the polling done i.e. WaitGroup counter is zero, and then close the channel
		p.waitGroup.Wait()
		close(p.opChan)
	}()

	for res := range p.opChan {
		if res.err != nil {
			p.log.Error(res.err, "operation failed", "opId", res.opId)
			err = common.Wrap(err, res.err)
		} else {
			p.log.Info("operations completed successfully", "opId", res.opId)
		}
	}

	return err
}
