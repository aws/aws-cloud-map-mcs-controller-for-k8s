package cloudmap

import (
	"context"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ServiceDiscoveryApi handles the AWS Cloud Map API request and response processing logic, and converts results to
// internal data structures. It manages all interactions with the AWS SDK.
type ServiceDiscoveryApi interface {
	// ListNamespaces returns a list of all namespaces.
	ListNamespaces(ctx context.Context) (namespaces []*Resource, err error)

	// ListServices returns a list of services for a given namespace.
	ListServices(ctx context.Context, namespaceId string) (services []*Resource, err error)

	// ListInstances returns a list of service instances registered to a given service.
	ListInstances(ctx context.Context, serviceId string) ([]*model.Endpoint, error)

	// ListOperations returns a map of operations to their status matching a list of filters.
	ListOperations(ctx context.Context, opFilters []types.OperationFilter) (operationStatusMap map[string]types.OperationStatus, err error)

	// GetOperation returns an operation.
	GetOperation(ctx context.Context, operationId string) (operation *types.Operation, err error)

	// CreateHttpNamespace creates a HTTP namespace in AWS Cloud Map for a given name.
	CreateHttpNamespace(ctx context.Context, namespaceName string) (operationId string, err error)

	// CreateService creates a named service in AWS Cloud Map under the given namespace.
	CreateService(ctx context.Context, namespaceId string, serviceName string) (serviceId string, err error)

	// RegisterInstance registers a service instance in AWS Cloud Map.
	RegisterInstance(ctx context.Context, serviceId string, instanceId string, instanceAttrs map[string]string) (operationId string, err error)

	// DeregisterInstance de-registers a service instance in Cloud Map.
	DeregisterInstance(ctx context.Context, serviceId string, instanceId string) (operationId string, err error)

	// PollCreateNamespace polls a create namespace operation, and returns the namespace ID.
	PollCreateNamespace(ctx context.Context, operationId string) (namespaceId string, err error)
}

type serviceDiscoveryApi struct {
	log       logr.Logger
	awsFacade AwsFacade
}

// Resource encapsulates a ID/name pair
type Resource struct {
	Id   string
	Name string
}

// NewServiceDiscoveryApiFromConfig creates a new AWS Cloud Map API connection manager from an AWS client config.
func NewServiceDiscoveryApiFromConfig(cfg *aws.Config) ServiceDiscoveryApi {
	return &serviceDiscoveryApi{
		log:       ctrl.Log.WithName("cloudmap"),
		awsFacade: NewAwsFacadeFromConfig(cfg),
	}
}

func (sdApi *serviceDiscoveryApi) ListNamespaces(ctx context.Context) ([]*Resource, error) {
	namespaces := make([]*Resource, 0)
	pages := sd.NewListNamespacesPaginator(sdApi.awsFacade, &sd.ListNamespacesInput{})

	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)
		if err != nil {
			return namespaces, err
		}

		for _, ns := range output.Namespaces {
			namespaces = append(namespaces, &Resource{
				Id:   aws.ToString(ns.Id),
				Name: aws.ToString(ns.Name),
			})
		}
	}

	return namespaces, nil
}

func (sdApi *serviceDiscoveryApi) ListServices(ctx context.Context, nsId string) ([]*Resource, error) {
	svcs := make([]*Resource, 0)

	filter := types.ServiceFilter{
		Name:   types.ServiceFilterNameNamespaceId,
		Values: []string{nsId},
	}
	sdApi.log.Info("paginating", "nsId", nsId)

	pages := sd.NewListServicesPaginator(sdApi.awsFacade, &sd.ListServicesInput{Filters: []types.ServiceFilter{filter}})

	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)
		if err != nil {
			return svcs, err
		}

		for _, svc := range output.Services {
			svcs = append(svcs, &Resource{
				Id:   aws.ToString(svc.Id),
				Name: aws.ToString(svc.Name),
			})
		}
	}

	return svcs, nil
}

func (sdApi *serviceDiscoveryApi) ListInstances(ctx context.Context, svcId string) ([]*model.Endpoint, error) {
	endpts := make([]*model.Endpoint, 0)

	pages := sd.NewListInstancesPaginator(sdApi.awsFacade, &sd.ListInstancesInput{ServiceId: &svcId})

	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)
		if err != nil {
			return endpts, err
		}

		for _, inst := range output.Instances {
			endpt, endptErr := model.NewEndpointFromInstance(&inst)

			if endptErr != nil {
				sdApi.log.Info(fmt.Sprintf("skipping instance %s to endpoint conversion: %s", *inst.Id, endptErr.Error()))
				continue
			}

			endpts = append(endpts, endpt)
		}
	}

	return endpts, nil
}

func (sdApi *serviceDiscoveryApi) ListOperations(ctx context.Context, opFilters []types.OperationFilter) (opStatusMap map[string]types.OperationStatus, err error) {
	opStatusMap = make(map[string]types.OperationStatus, 0)

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

func (sdApi *serviceDiscoveryApi) CreateService(ctx context.Context, nsId string, svcName string) (svcId string, err error) {
	output, err := sdApi.awsFacade.CreateService(ctx, &sd.CreateServiceInput{
		NamespaceId: &nsId,
		Name:        &svcName})

	if err != nil {
		return "", err
	}

	svcId = aws.ToString(output.Service.Id)
	sdApi.log.Info("service created", "svcId", svcId)
	return svcId, nil
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

func (sdApi *serviceDiscoveryApi) PollCreateNamespace(ctx context.Context, opId string) (nsId string, err error) {
	return nsId, wait.Poll(defaultOperationPollInterval, defaultOperationPollTimeout, func() (done bool, pollErr error) {
		sdApi.log.Info("polling operation", "opId", opId)
		op, opErr := sdApi.GetOperation(ctx, opId)

		if opErr != nil {
			return true, opErr
		}

		if op.Status == types.OperationStatusFail {
			return true, fmt.Errorf("failed to create namespace: %s", aws.ToString(op.ErrorMessage))
		}

		if op.Status == types.OperationStatusSuccess {
			nsId = op.Targets[string(types.OperationTargetTypeNamespace)]
			sdApi.log.Info("namespace created", "nsId", nsId)
			return true, nil
		}

		return false, nil
	})
}
