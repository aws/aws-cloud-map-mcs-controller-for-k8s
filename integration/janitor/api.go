package janitor

import (
	"context"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
)

type ServiceDiscoveryJanitorApi interface {
	DeleteNamespace(ctx context.Context, namespaceId string) (operationId string, err error)
	DeleteService(ctx context.Context, serviceId string) error
	cloudmap.ServiceDiscoveryApi
}

type serviceDiscoveryJanitorApi struct {
	cloudmap.ServiceDiscoveryApi
	janitorFacade SdkJanitorFacade
}

func NewServiceDiscoveryJanitorApiFromConfig(cfg *aws.Config) ServiceDiscoveryJanitorApi {
	return &serviceDiscoveryJanitorApi{
		ServiceDiscoveryApi: cloudmap.NewServiceDiscoveryApiFromConfig(cfg),
		janitorFacade:       NewSdkJanitorFacadeFromConfig(cfg),
	}
}

func (api *serviceDiscoveryJanitorApi) DeleteNamespace(ctx context.Context, nsId string) (opId string, err error) {
	out, err := api.janitorFacade.DeleteNamespace(ctx, &sd.DeleteNamespaceInput{Id: &nsId})
	if err != nil {
		return "", err
	}

	return aws.ToString(out.OperationId), nil
}

func (api *serviceDiscoveryJanitorApi) DeleteService(ctx context.Context, svcId string) error {
	_, err := api.janitorFacade.DeleteService(ctx, &sd.DeleteServiceInput{Id: &svcId})
	return err
}
