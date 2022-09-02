package cloudmap

import (
	"context"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/version"
	"github.com/aws/aws-sdk-go-v2/aws/middleware"

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

	// DiscoverInstances provides ServiceDiscovery DiscoverInstances wrapper interface.
	DiscoverInstances(context.Context, *sd.DiscoverInstancesInput, ...func(*sd.Options)) (*sd.DiscoverInstancesOutput, error)
}

type awsFacade struct {
	*sd.Client
}

// NewAwsFacadeFromConfig creates a new AWS facade from an AWS client config.
func NewAwsFacadeFromConfig(cfg *aws.Config) AwsFacade {
	sdClient := sd.NewFromConfig(*cfg, func(options *sd.Options) {
		// Append User-Agent to all the request, the format is going to be aws-cloud-map-mcs-controller-for-k8s/0.0.0-abc
		options.APIOptions = append(options.APIOptions, middleware.AddUserAgentKeyValue(version.GetUserAgentKey(), version.GetUserAgentValue()))
	})
	return &awsFacade{sdClient}
}
