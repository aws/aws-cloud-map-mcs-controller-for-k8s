package cloudmap

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
)

type AwsFacade interface {
	ListNamespaces(context.Context, *sd.ListNamespacesInput, ...func(*sd.Options)) (*sd.ListNamespacesOutput, error)
	ListServices(context.Context, *sd.ListServicesInput, ...func(options *sd.Options)) (*sd.ListServicesOutput, error)
	ListInstances(context.Context, *sd.ListInstancesInput, ...func(*sd.Options)) (*sd.ListInstancesOutput, error)
	ListOperations(context.Context, *sd.ListOperationsInput, ...func(*sd.Options)) (*sd.ListOperationsOutput, error)
	GetOperation(context.Context, *sd.GetOperationInput, ...func(*sd.Options)) (*sd.GetOperationOutput, error)
	CreateHttpNamespace(context.Context, *sd.CreateHttpNamespaceInput, ...func(*sd.Options)) (*sd.CreateHttpNamespaceOutput, error)
	CreateService(context.Context, *sd.CreateServiceInput, ...func(*sd.Options)) (*sd.CreateServiceOutput, error)
	RegisterInstance(context.Context, *sd.RegisterInstanceInput, ...func(*sd.Options)) (*sd.RegisterInstanceOutput, error)
	DeregisterInstance(context.Context, *sd.DeregisterInstanceInput, ...func(*sd.Options)) (*sd.DeregisterInstanceOutput, error)
}

type awsFacade struct {
	*sd.Client
}

func NewAwsFacadeFromConfig(cfg *aws.Config) AwsFacade {
	return &awsFacade{sd.NewFromConfig(*cfg)}
}
