package cloudmap

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-k8s-mcs-controller/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"time"
)

const (
	defaultEndpointsCacheTTL  = 5 * time.Second
	defaultEndpointsCacheSize = 1024

	defaultOperationPollInterval = 3 * time.Second
)

type EndpointManager interface {
	ListEndpoints(ctx context.Context, serviceId string) ([]*model.Endpoint, error)

	RegisterEndpoints(ctx context.Context, service *model.Service, serviceId string) error

	DeregisterEndpoints(ctx context.Context, service *model.Service, serviceId string) error
}

type defaultEndpointManager struct {
	log              logr.Logger
	sdApi            *sd.Client
	endpointCache    *cache.LRUExpireCache
	endpointCacheTTL time.Duration

	// interval between each getOperation call
	operationPollInterval time.Duration
	// maximum retries per getOperation call
	operationPollMaxRetries int
}

func NewEndpointManager(cfg *aws.Config) EndpointManager {
	return &defaultEndpointManager{
		log:                   ctrl.Log.WithName("cloudmap"),
		sdApi:                 sd.NewFromConfig(*cfg),
		endpointCache:         cache.NewLRUExpireCache(defaultEndpointsCacheSize),
		endpointCacheTTL:      defaultEndpointsCacheTTL,
		operationPollInterval: defaultOperationPollInterval,
	}
}

func (mgr *defaultEndpointManager) ListEndpoints(ctx context.Context, serviceId string) ([]*model.Endpoint, error) {

	if cachedValue, exists := mgr.endpointCache.Get(serviceId); exists {
		return cachedValue.([]*model.Endpoint), nil
	}

	endpts, endptsErr := mgr.listEndpointsFromCloudMap(ctx, serviceId)

	if endptsErr != nil {
		return nil, endptsErr
	}

	mgr.endpointCache.Add(serviceId, endpts, defaultEndpointsCacheTTL)

	return endpts, nil
}

func (mgr *defaultEndpointManager) listEndpointsFromCloudMap(ctx context.Context, svcId string) ([]*model.Endpoint, error) {
	endpts := make([]*model.Endpoint, 0)

	pages := sd.NewListInstancesPaginator(mgr.sdApi, &sd.ListInstancesInput{ServiceId: &svcId})

	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)
		if err != nil {
			return endpts, err
		}

		for _, inst := range output.Instances {
			endpt, endptErr := model.NewEndpointFromInstance(&inst)

			if endptErr != nil {
				mgr.log.Info(fmt.Sprintf("skipping instance %s to endpoint conversion: %s", *inst.Id, endptErr.Error()))
				continue
			}

			endpts = append(endpts, endpt)
		}
	}

	return endpts, nil
}

type opResult struct {
	instId string
	opId   string
	err    error
}

func (mgr *defaultEndpointManager) RegisterEndpoints(ctx context.Context, service *model.Service, serviceId string) error {
	opChan := make(chan opResult)
	startTime := mgr.now()

	for _, endpt := range service.Endpoints {
		go mgr.registerInstanceInCloudMap(ctx, serviceId, endpt.Id, endpt.GetAttributes(), opChan)
	}

	ops, regSuccess := mgr.getOpsList(len(service.Endpoints), opChan)
	opsErr := mgr.pollOperations(ctx, types.OperationTypeRegisterInstance, serviceId, startTime, ops)

	// Evict cache entry so next list call reflects changes
	mgr.endpointCache.Remove(serviceId)

	if opsErr != nil {
		return opsErr
	}

	if !regSuccess {
		return errors.New("failure registering endpoints")
	}

	return nil
}

func (mgr *defaultEndpointManager) registerInstanceInCloudMap(ctx context.Context, svcId string, instId string, instAttrs map[string]string, opChan chan opResult) {

	regResp, err := mgr.sdApi.RegisterInstance(ctx, &sd.RegisterInstanceInput{
		Attributes: instAttrs,
		InstanceId: &instId,
		ServiceId:  &svcId,
	})

	opChan <- opResult{instId, aws.ToString(regResp.OperationId), err}
}

func (mgr *defaultEndpointManager) DeregisterEndpoints(ctx context.Context, service *model.Service, serviceId string) error {
	opChan := make(chan opResult)
	startTime := mgr.now()

	for _, endpt := range service.Endpoints {
		go mgr.deregisterInstanceInCloudMap(ctx, serviceId, endpt.Id, opChan)
	}

	ops, deregSuccess := mgr.getOpsList(len(service.Endpoints), opChan)

	opsErr := mgr.pollOperations(ctx, types.OperationTypeDeregisterInstance, serviceId, startTime, ops)

	// Evict cache entry so next list call reflects changes
	mgr.endpointCache.Remove(serviceId)

	if opsErr != nil {
		return opsErr
	}

	if !deregSuccess {
		return errors.New("failure de-registering endpoints")
	}

	return nil
}

func (mgr *defaultEndpointManager) deregisterInstanceInCloudMap(ctx context.Context, svcId string, instId string, opChan chan opResult) {
	deregResp, err := mgr.sdApi.DeregisterInstance(ctx, &sd.DeregisterInstanceInput{
		InstanceId: &instId,
		ServiceId:  &svcId,
	})

	if err != nil {
		opChan <- opResult{instId, "", err}
	}

	opChan <- opResult{instId, *deregResp.OperationId, nil}
}

func (mgr *defaultEndpointManager) getOpsList(opCount int, opChan chan opResult) ([]string, bool) {
	success := true

	ops := make([]string, 0)

	for i := 0; i < opCount; i++ {
		op := <-opChan

		if op.err != nil {
			mgr.log.Info("could not create operation", "error", op.err)
			success = false
			continue
		}

		ops = append(ops, op.opId)
	}

	return ops, success
}

func (mgr *defaultEndpointManager) pollOperations(ctx context.Context, opType types.OperationType, svcId string, startTime int64, ops []string) error {

	if len(ops) == 0 {
		mgr.log.Info("no operations to poll")
	}

	svcFilter := types.OperationFilter{
		Name:   types.OperationFilterNameServiceId,
		Values: []string{svcId},
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
		Values: []string{string(opType)},
	}

	timeFilter := types.OperationFilter{
		Name:      types.OperationFilterNameUpdateDate,
		Condition: types.FilterConditionBetween,
		Values: []string{
			strconv.Itoa(int(startTime)),
			// Add one minute to end range in case op updates while list request is in flight
			strconv.Itoa(int(mgr.now() + 60000)),
		},
	}

	return wait.PollUntil(mgr.operationPollInterval, func() (bool, error) {
		mgr.log.Info("polling operations", "operations", ops)
		completed := 0
		failedOps := make([]string, 0)

		pages := sd.NewListOperationsPaginator(mgr.sdApi, &sd.ListOperationsInput{
			Filters: []types.OperationFilter{svcFilter, statusFilter, typeFilter, timeFilter},
		})

		for pages.HasMorePages() {
			output, err := pages.NextPage(ctx)

			if err != nil {
				return true, err
			}

			for _, pollOp := range ops {
				for _, sdOp := range output.Operations {
					if pollOp == aws.ToString(sdOp.Id) {
						completed++
						if sdOp.Status == types.OperationStatusFail {
							failedOps = append(failedOps, pollOp)
						}
					}
				}
			}
		}

		if completed != len(ops) {
			return false, nil
		}

		if len(failedOps) != 0 {
			for _, failedOp := range failedOps {
				mgr.log.Info("Operation failed", "failedOp", failedOp, "reason", mgr.getFailedOpReason(ctx, failedOp))
			}
			return true, fmt.Errorf("operation failure")
		}

		mgr.log.Info("operations completed successfully")
		return true, nil
	}, ctx.Done())
}

// getFailedOpReason returns operation error message, which is not available in ListOperations response
func (mgr *defaultEndpointManager) getFailedOpReason(ctx context.Context, op string) string {
	opResp, err := mgr.sdApi.GetOperation(ctx, &sd.GetOperationInput{OperationId: &op})

	if err != nil {
		return "failed to retrieve operation failure reason"
	}

	return aws.ToString(opResp.Operation.ErrorMessage)
}

// now returns current time with milliseconds, as used by operation UPDATE_DATE field
func (mgr *defaultEndpointManager) now() int64 {
	return time.Now().UnixNano() / 1000000
}
