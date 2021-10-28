package cloudmap

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
)

// AwsFacade wraps the minimal surface area of ServiceDiscovery API calls for the AWS SDK
// required by the AWS Cloud Map client. This enables mock generation for unit testing.
type AwsFacade interface {
	// ListNamespaces provides ServiceDiscovery ListNamespaces wrapper interface for paginator.
	ListNamespaces(context.Context, *sd.ListNamespacesInput, ...func(*sd.Options)) (*sd.ListNamespacesOutput, error)

	// ListServices provides ServiceDiscovery ListServices wrapper interface for paginator.
	ListServices(context.Context, *sd.ListServicesInput, ...func(options *sd.Options)) (*sd.ListServicesOutput, error)

	// ListInstances provides ServiceDiscovery ListInstances wrapper interface for paginator.
	ListInstances(context.Context, *sd.ListInstancesInput, ...func(*sd.Options)) (*sd.ListInstancesOutput, error)

	// ListOperations provides ServiceDiscovery ListOperations wrapper interface for paginator.
	ListOperations(context.Context, *sd.ListOperationsInput, ...func(*sd.Options)) (*sd.ListOperationsOutput, error)

	// GetOperation provides ServiceDiscovery GetOperation wrapper interface.
	GetOperation(context.Context, *sd.GetOperationInput, ...func(*sd.Options)) (*sd.GetOperationOutput, error)

	// CreateHttpNamespace provides ServiceDiscovery CreateHttpNamespace wrapper interface.
	CreateHttpNamespace(context.Context, *sd.CreateHttpNamespaceInput, ...func(*sd.Options)) (*sd.CreateHttpNamespaceOutput, error)

	// CreateService provides ServiceDiscovery CreateService wrapper interface.
	CreateService(context.Context, *sd.CreateServiceInput, ...func(*sd.Options)) (*sd.CreateServiceOutput, error)

	// RegisterInstance provides ServiceDiscovery RegisterInstance wrapper interface.
	RegisterInstance(context.Context, *sd.RegisterInstanceInput, ...func(*sd.Options)) (*sd.RegisterInstanceOutput, error)

	// DeregisterInstance provides ServiceDiscovery DeregisterInstance wrapper interface.
	DeregisterInstance(context.Context, *sd.DeregisterInstanceInput, ...func(*sd.Options)) (*sd.DeregisterInstanceOutput, error)

	// DeleteNamespace provides ServiceDiscovery DeleteNamespace wrapper interface.
	DeleteNamespace(context.Context, *sd.DeleteNamespaceInput, ...func(*sd.Options)) (*sd.DeleteNamespaceOutput, error)

	// DeleteService provides ServiceDiscovery DeleteService wrapper interface.
	DeleteService(context.Context, *sd.DeleteServiceInput, ...func(*sd.Options)) (*sd.DeleteServiceOutput, error)
}

type awsFacade struct {
	*sd.Client
}

// NewAwsFacadeFromConfig creates a new AWS facade from an AWS client config.
func NewAwsFacadeFromConfig(cfg *aws.Config) AwsFacade {
	return &awsFacade{sd.NewFromConfig(*cfg)}
}
