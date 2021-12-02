package janitor

import (
	"context"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
)

// SdkJanitorFacade extends the minimal surface area of ServiceDiscovery API calls of the client
// for integration test janitor operations.
type SdkJanitorFacade interface {
	// DeleteNamespace provides ServiceDiscovery DeleteNamespace wrapper interface.
	DeleteNamespace(context.Context, *sd.DeleteNamespaceInput, ...func(*sd.Options)) (*sd.DeleteNamespaceOutput, error)

	// DeleteService provides ServiceDiscovery DeleteService wrapper interface.
	DeleteService(context.Context, *sd.DeleteServiceInput, ...func(*sd.Options)) (*sd.DeleteServiceOutput, error)

	cloudmap.AwsFacade
}

type sdkJanitorFacade struct {
	*sd.Client
}

// NewSdkJanitorFacadeFromConfig creates a new AWS facade from an AWS client config
// extended for integration test janitor operations.
func NewSdkJanitorFacadeFromConfig(cfg *aws.Config) SdkJanitorFacade {
	return &sdkJanitorFacade{sd.NewFromConfig(*cfg)}
}
