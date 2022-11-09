package cloudmap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/time/rate"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultServiceTTLInSeconds int64 = 60
)

// ServiceDiscoveryApi handles the AWS Cloud Map API request and response processing logic, and converts results to
// internal data structures. It manages all interactions with the AWS SDK.
type ServiceDiscoveryApi interface {
	// GetNamespaceMap returns a map of all namespaces in the Cloud Map account indexed by namespace name.
	GetNamespaceMap(ctx context.Context) (namespaces map[string]*model.Namespace, err error)

	// GetServiceIdMap returns a map of all service IDs for a given namespace indexed by service name.
	GetServiceIdMap(ctx context.Context, namespaceId string) (serviceIdMap map[string]string, err error)

	// DiscoverInstances returns a list of service instances registered to a given service.
	DiscoverInstances(ctx context.Context, nsName string, svcName string, queryParameters map[string]string) (insts []types.HttpInstanceSummary, err error)

	// ListOperations returns a map of operations to their status matching a list of filters.
	ListOperations(ctx context.Context, opFilters []types.OperationFilter) (operationStatusMap map[string]types.OperationStatus, err error)

	// GetOperation returns an operation.
	GetOperation(ctx context.Context, operationId string) (operation *types.Operation, err error)

	// CreateHttpNamespace creates a HTTP namespace in AWS Cloud Map for a given name.
	CreateHttpNamespace(ctx context.Context, namespaceName string) (operationId string, err error)

	// CreateService creates a named service in AWS Cloud Map under the given namespace.
	CreateService(ctx context.Context, namespace model.Namespace, serviceName string) (serviceId string, err error)

	// RegisterInstance registers a service instance in AWS Cloud Map.
	RegisterInstance(ctx context.Context, serviceId string, instanceId string, instanceAttrs map[string]string) (operationId string, err error)

	// DeregisterInstance de-registers a service instance in Cloud Map.
	DeregisterInstance(ctx context.Context, serviceId string, instanceId string) (operationId string, err error)

	// PollNamespaceOperation polls a namespace operation, and returns the namespace ID.
	PollNamespaceOperation(ctx context.Context, operationId string) (namespaceId string, err error)
}

type serviceDiscoveryApi struct {
	log            common.Logger
	awsFacade      AwsFacade
	nsRateLimiter  *rate.Limiter
	svcRateLimiter *rate.Limiter
	opRateLimiter  *rate.Limiter
}

// NewServiceDiscoveryApiFromConfig creates a new AWS Cloud Map API connection manager from an AWS client config.
func NewServiceDiscoveryApiFromConfig(cfg *aws.Config) ServiceDiscoveryApi {
	return &serviceDiscoveryApi{
		log:            common.NewLogger("cloudmap", "api"),
		awsFacade:      NewAwsFacadeFromConfig(cfg),
		nsRateLimiter:  rate.NewLimiter(rate.Every(1*time.Second), 5),     // 1 per second
		svcRateLimiter: rate.NewLimiter(rate.Every(2*time.Second), 10),    // 2 per second
		opRateLimiter:  rate.NewLimiter(rate.Every(100*time.Second), 200), // 100 per second
	}
}

func (sdApi *serviceDiscoveryApi) GetNamespaceMap(ctx context.Context) (map[string]*model.Namespace, error) {
	err := sdApi.nsRateLimiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	namespaceMap := make(map[string]*model.Namespace)

	pages := sd.NewListNamespacesPaginator(sdApi.awsFacade, &sd.ListNamespacesInput{})
	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, ns := range output.Namespaces {
			namespaceType := model.ConvertNamespaceType(ns.Type)
			if namespaceType.IsUnsupported() {
				continue
			}
			namespaceMap[aws.ToString(ns.Name)] = &model.Namespace{
				Id:   aws.ToString(ns.Id),
				Name: aws.ToString(ns.Name),
				Type: namespaceType,
			}
		}
	}

	return namespaceMap, nil
}

func (sdApi *serviceDiscoveryApi) GetServiceIdMap(ctx context.Context, nsId string) (map[string]string, error) {
	err := sdApi.svcRateLimiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	serviceIdMap := make(map[string]string)

	filter := types.ServiceFilter{
		Name:   types.ServiceFilterNameNamespaceId,
		Values: []string{nsId},
	}

	pages := sd.NewListServicesPaginator(sdApi.awsFacade, &sd.ListServicesInput{Filters: []types.ServiceFilter{filter}})
	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, svc := range output.Services {
			serviceIdMap[aws.ToString(svc.Name)] = aws.ToString(svc.Id)
		}
	}

	return serviceIdMap, nil
}

func (sdApi *serviceDiscoveryApi) DiscoverInstances(ctx context.Context, nsName string, svcName string, queryParameters map[string]string) (insts []types.HttpInstanceSummary, err error) {
	input := &sd.DiscoverInstancesInput{
		NamespaceName: aws.String(nsName),
		ServiceName:   aws.String(svcName),
		HealthStatus:  types.HealthStatusFilterAll,
		MaxResults:    aws.Int32(1000),
	}
	if queryParameters != nil {
		input.QueryParameters = queryParameters
	}
	out, err := sdApi.awsFacade.DiscoverInstances(ctx, input)

	if err != nil {
		return insts, err
	}

	return out.Instances, nil
}

func (sdApi *serviceDiscoveryApi) ListOperations(ctx context.Context, opFilters []types.OperationFilter) (map[string]types.OperationStatus, error) {
	opStatusMap := make(map[string]types.OperationStatus)

	pages := sd.NewListOperationsPaginator(sdApi.awsFacade, &sd.ListOperationsInput{
		Filters: opFilters,
	})

	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)

		if err != nil {
			return opStatusMap, err
		}

		for _, sdOp := range output.Operations {
			opStatusMap[aws.ToString(sdOp.Id)] = sdOp.Status
		}
	}

	return opStatusMap, nil
}

func (sdApi *serviceDiscoveryApi) GetOperation(ctx context.Context, opId string) (operation *types.Operation, err error) {
	err = sdApi.opRateLimiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	opResp, err := sdApi.awsFacade.GetOperation(ctx, &sd.GetOperationInput{OperationId: &opId})

	if err != nil {
		return nil, err
	}

	return opResp.Operation, nil
}

func (sdApi *serviceDiscoveryApi) CreateHttpNamespace(ctx context.Context, nsName string) (opId string, err error) {
	output, err := sdApi.awsFacade.CreateHttpNamespace(ctx, &sd.CreateHttpNamespaceInput{
		Name: &nsName,
	})

	if err != nil {
		return "", err
	}

	return aws.ToString(output.OperationId), nil
}

func (sdApi *serviceDiscoveryApi) CreateService(ctx context.Context, namespace model.Namespace, svcName string) (svcId string, err error) {
	var output *sd.CreateServiceOutput
	if namespace.Type == model.DnsPrivateNamespaceType {
		dnsConfig := sdApi.getDnsConfig()
		output, err = sdApi.awsFacade.CreateService(ctx, &sd.CreateServiceInput{
			NamespaceId: &namespace.Id,
			DnsConfig:   &dnsConfig,
			Name:        &svcName})
	} else {
		output, err = sdApi.awsFacade.CreateService(ctx, &sd.CreateServiceInput{
			NamespaceId: &namespace.Id,
			Name:        &svcName})
	}

	if err != nil {
		return "", err
	}

	svcId = aws.ToString(output.Service.Id)
	sdApi.log.Info("service created", "namespace", namespace.Name, "name", svcName, "id", svcId)
	return svcId, nil
}

func (sdApi *serviceDiscoveryApi) getDnsConfig() types.DnsConfig {
	dnsConfig := types.DnsConfig{
		DnsRecords: []types.DnsRecord{
			{
				TTL:  aws.Int64(defaultServiceTTLInSeconds),
				Type: "SRV",
			},
		},
	}
	return dnsConfig
}

func (sdApi *serviceDiscoveryApi) RegisterInstance(ctx context.Context, svcId string, instId string, instAttrs map[string]string) (opId string, err error) {
	regResp, err := sdApi.awsFacade.RegisterInstance(ctx, &sd.RegisterInstanceInput{
		Attributes: instAttrs,
		InstanceId: &instId,
		ServiceId:  &svcId,
	})

	if err != nil {
		return "", err
	}

	return aws.ToString(regResp.OperationId), nil
}

func (sdApi *serviceDiscoveryApi) DeregisterInstance(ctx context.Context, svcId string, instId string) (opId string, err error) {
	deregResp, err := sdApi.awsFacade.DeregisterInstance(ctx, &sd.DeregisterInstanceInput{
		InstanceId: &instId,
		ServiceId:  &svcId,
	})

	if err != nil {
		return "", err
	}

	return aws.ToString(deregResp.OperationId), err
}

func (sdApi *serviceDiscoveryApi) PollNamespaceOperation(ctx context.Context, opId string) (nsId string, err error) {
	err = wait.Poll(defaultOperationPollInterval, defaultOperationPollTimeout, func() (done bool, err error) {
		sdApi.log.Info("polling operation", "opId", opId)
		op, err := sdApi.GetOperation(ctx, opId)

		if err != nil {
			return true, err
		}

		if op.Status == types.OperationStatusFail {
			return true, fmt.Errorf("failed to create namespace: %s", aws.ToString(op.ErrorMessage))
		}

		if op.Status == types.OperationStatusSuccess {
			nsId = op.Targets[string(types.OperationTargetTypeNamespace)]
			return true, nil
		}

		return false, nil
	})

	if err == wait.ErrWaitTimeout {
		err = errors.New(operationPollTimoutErrorMessage)
	}

	return nsId, err
}
