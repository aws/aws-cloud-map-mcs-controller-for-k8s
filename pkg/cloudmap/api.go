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

type ServiceDiscoveryApi interface {
	ListServices(ctx context.Context, namespaceId string) (services []*Resource, err error)
	ListInstances(ctx context.Context, serviceId string) ([]*model.Endpoint, error)
	ListOperations(ctx context.Context, opFilters []types.OperationFilter) (operationStatusMap map[string]types.OperationStatus, err error)
	GetNamespaceId(ctx context.Context, namespaceName string) (namespaceId string, err error)
	GetOperationErrorMessage(ctx context.Context, operationId string) (operationError string, err error)
	CreateHttpNamespace(ctx context.Context, namespaceName string) (operationId string, err error)
	CreateService(ctx context.Context, namespaceId string, serviceName string) (serviceId string, err error)
	RegisterInstance(ctx context.Context, serviceId string, instanceId string, instanceAttrs map[string]string) (operationId string, err error)
	DeregisterInstance(ctx context.Context, serviceId string, instanceId string) (operationId string, err error)
	PollCreateNamespace(ctx context.Context, operationId string) (namespaceId string, err error)
}

type serviceDiscoveryApi struct {
	log       logr.Logger
	awsFacade AwsFacade
}

type Resource struct {
	Id   string
	Name string
}

func NewServiceDiscoveryApiFromConfig(cfg *aws.Config) ServiceDiscoveryApi {
	return &serviceDiscoveryApi{
		log:       ctrl.Log.WithName("cloudmap"),
		awsFacade: NewAwsFacadeFromConfig(cfg),
	}
}

func (sdApi *serviceDiscoveryApi) ListServices(ctx context.Context, nsId string) ([]*Resource, error) {
	svcs := make([]*Resource, 0)

	filter := types.ServiceFilter{
		Name:   types.ServiceFilterNameNamespaceId,
		Values: []string{nsId},
	}

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

func (sdApi *serviceDiscoveryApi) GetNamespaceId(ctx context.Context, nsName string) (nsID string, err error) {
	pages := sd.NewListNamespacesPaginator(sdApi.awsFacade, &sd.ListNamespacesInput{})

	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)
		if err != nil {
			return "", err
		}

		for _, ns := range output.Namespaces {
			if nsName == aws.ToString(ns.Name) {
				return aws.ToString(ns.Id), nil
			}
		}
	}

	return "", fmt.Errorf("namespace %s not found", nsName)
}

func (sdApi *serviceDiscoveryApi) GetOperationErrorMessage(ctx context.Context, opId string) (operationError string, err error) {
	opResp, err := sdApi.awsFacade.GetOperation(ctx, &sd.GetOperationInput{OperationId: &opId})

	if err != nil {
		return "", err
	}

	return aws.ToString(opResp.Operation.ErrorMessage), nil
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

func (sdApi *serviceDiscoveryApi) CreateService(ctx context.Context, nsId string, svcName string) (opId string, err error) {
	output, err := sdApi.awsFacade.CreateService(ctx, &sd.CreateServiceInput{
		NamespaceId: &nsId,
		Name:        &svcName})

	if err != nil {
		return "", err
	}

	return aws.ToString(output.Service.Id), nil
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
	err = wait.PollUntil(defaultOperationPollInterval, func() (done bool, pollErr error) {
		opResult, pollErr := sdApi.awsFacade.GetOperation(ctx, &sd.GetOperationInput{
			OperationId: &opId,
		})

		if pollErr != nil {
			return true, pollErr
		}

		if opResult.Operation.Status == types.OperationStatusFail {
			return true, fmt.Errorf("failed to create namespace.Reason: %s", *opResult.Operation.ErrorMessage)
		}

		if opResult.Operation.Status == types.OperationStatusSuccess {
			nsId = opResult.Operation.Targets[string(types.OperationTargetTypeNamespace)]
			return true, nil
		}

		return false, nil
	}, ctx.Done())

	if err != nil {
		return "", err
	}

	return nsId, nil
}