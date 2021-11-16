package cloudmap

import (
	"context"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ServiceDiscoveryClient provides the service endpoint management functionality required by the AWS Cloud Map
// multi-cluster service discovery for Kubernetes controller. It maintains local caches for all AWS Cloud Map resources.
type ServiceDiscoveryClient interface {
	// ListServices returns all services and their endpoints for a given namespace.
	ListServices(ctx context.Context, namespaceName string) ([]*model.Service, error)

	// CreateService creates a Cloud Map service resource, and namespace if necessary.
	CreateService(ctx context.Context, namespaceName string, serviceName string) error

	// GetService returns a service resource fetched from AWS Cloud Map or nil if not found.
	GetService(ctx context.Context, namespaceName string, serviceName string) (*model.Service, error)

	// RegisterEndpoints registers all endpoints for given service.
	RegisterEndpoints(ctx context.Context, namespaceName string, serviceName string, endpoints []*model.Endpoint) error

	// DeleteEndpoints de-registers all endpoints for given service.
	DeleteEndpoints(ctx context.Context, namespaceName string, serviceName string, endpoints []*model.Endpoint) error
}

type serviceDiscoveryClient struct {
	log   logr.Logger
	sdApi ServiceDiscoveryApi
	cache ServiceDiscoveryClientCache
}

// NewDefaultServiceDiscoveryClient creates a new service discovery client for AWS Cloud Map with default resource cache
// from a given AWS client config.
func NewDefaultServiceDiscoveryClient(cfg *aws.Config) ServiceDiscoveryClient {
	return &serviceDiscoveryClient{
		log:   ctrl.Log.WithName("cloudmap"),
		sdApi: NewServiceDiscoveryApiFromConfig(cfg),
		cache: NewDefaultServiceDiscoveryClientCache(),
	}
}

func NewServiceDiscoveryClientWithCustomCache(cfg *aws.Config, cacheConfig *SdCacheConfig) ServiceDiscoveryClient {
	return &serviceDiscoveryClient{
		log:   ctrl.Log.WithName("cloudmap"),
		sdApi: NewServiceDiscoveryApiFromConfig(cfg),
		cache: NewServiceDiscoveryClientCache(cacheConfig),
	}
}

func (sdc *serviceDiscoveryClient) ListServices(ctx context.Context, nsName string) (svcs []*model.Service, err error) {
	namespace, err := sdc.getNamespace(ctx, nsName)
	if err != nil || namespace == nil {
		return svcs, err
	}

	// TODO: Cache list
	svcSums, err := sdc.sdApi.ListServices(ctx, namespace.Id)
	if err != nil {
		return svcs, err
	}

	for _, svcSum := range svcSums {
		sdc.cache.CacheServiceId(nsName, svcSum.Name, svcSum.Id)

		endpts, endptsErr := sdc.listEndpoints(ctx, nsName, svcSum.Name)
		if endptsErr != nil {
			return svcs, endptsErr
		}

		svcs = append(svcs, &model.Service{
			Namespace: nsName,
			Name:      svcSum.Name,
			Endpoints: endpts,
		})
	}

	return svcs, nil
}

func (sdc *serviceDiscoveryClient) CreateService(ctx context.Context, nsName string, svcName string) (err error) {
	sdc.log.Info("creating a new service", "namespace", nsName, "name", svcName)

	namespace, err := sdc.getNamespace(ctx, nsName)
	if err != nil {
		return err
	}

	if namespace == nil {
		// Create HttpNamespace if the namespace is not present in the CloudMap
		namespace, err = sdc.createNamespace(ctx, nsName)
		if err != nil {
			return err
		}
	}

	svcId, err := sdc.sdApi.CreateService(ctx, *namespace, svcName)
	if err != nil {
		return err
	}

	sdc.cache.CacheServiceId(nsName, svcName, svcId)

	return nil
}

func (sdc *serviceDiscoveryClient) GetService(ctx context.Context, nsName string, svcName string) (svc *model.Service, err error) {
	sdc.log.Info("fetching a service", "namespace", nsName, "name", svcName)
	endpts, cacheHit := sdc.cache.GetEndpoints(nsName, svcName)

	if cacheHit {
		return &model.Service{
			Namespace: nsName,
			Name:      svcName,
			Endpoints: endpts,
		}, nil
	}

	svcId, err := sdc.getServiceId(ctx, nsName, svcName)

	if err != nil {
		return nil, err
	}

	if svcId == "" {
		return nil, nil
	}

	endpts, err = sdc.listEndpoints(ctx, nsName, svcName)

	if err != nil {
		return nil, err
	}

	return &model.Service{
		Namespace: nsName,
		Name:      svcName,
		Endpoints: endpts,
	}, nil
}

func (sdc *serviceDiscoveryClient) RegisterEndpoints(ctx context.Context, nsName string, svcName string, endpts []*model.Endpoint) (err error) {
	if len(endpts) == 0 {
		sdc.log.Info("skipping endpoint registration for empty endpoint list", "serviceName", svcName)
		return nil
	}

	sdc.log.Info("registering endpoints", "namespaceName", nsName, "serviceName", svcName, "endpoints", endpts)

	svcId, err := sdc.getServiceId(ctx, nsName, svcName)
	if err != nil {
		return err
	}

	opCollector := NewOperationCollector()

	for _, endpt := range endpts {
		endptId := endpt.Id
		endptAttrs := endpt.GetCloudMapAttributes()
		opCollector.Add(func() (opId string, err error) {
			return sdc.sdApi.RegisterInstance(ctx, svcId, endptId, endptAttrs)
		})
	}

	err = NewRegisterInstancePoller(sdc.sdApi, svcId, opCollector.Collect(), opCollector.GetStartTime()).Poll(ctx)

	// Evict cache entry so next list call reflects changes
	sdc.cache.EvictEndpoints(nsName, svcName)

	if err != nil {
		return err
	}

	if !opCollector.IsAllOperationsCreated() {
		return fmt.Errorf("failure while registering endpoints")
	}

	return nil
}

func (sdc *serviceDiscoveryClient) DeleteEndpoints(ctx context.Context, nsName string, svcName string, endpts []*model.Endpoint) (err error) {
	if len(endpts) == 0 {
		sdc.log.Info("skipping endpoint deletion for empty endpoint list", "serviceName", svcName)
		return nil
	}

	sdc.log.Info("deleting endpoints", "namespaceName", nsName,
		"serviceName", svcName, "endpoints", endpts)

	svcId, err := sdc.getServiceId(ctx, nsName, svcName)
	if err != nil {
		return err
	}

	opCollector := NewOperationCollector()

	for _, endpt := range endpts {
		endptId := endpt.Id
		opCollector.Add(func() (opId string, err error) {
			return sdc.sdApi.DeregisterInstance(ctx, svcId, endptId)
		})
	}

	err = NewDeregisterInstancePoller(sdc.sdApi, svcId, opCollector.Collect(), opCollector.GetStartTime()).Poll(ctx)

	// Evict cache entry so next list call reflects changes
	sdc.cache.EvictEndpoints(nsName, svcName)
	if err != nil {
		return err
	}

	if !opCollector.IsAllOperationsCreated() {
		return fmt.Errorf("failure while de-registering endpoints")
	}

	return nil
}

func (sdc *serviceDiscoveryClient) listEndpoints(ctx context.Context, nsName string, svcName string) (endpts []*model.Endpoint, err error) {
	if endpts, found := sdc.cache.GetEndpoints(nsName, svcName); found {
		return endpts, nil
	}

	insts, err := sdc.sdApi.DiscoverInstances(ctx, nsName, svcName)
	if err != nil {
		return nil, err
	}

	for _, inst := range insts {
		endpt, endptErr := model.NewEndpointFromInstance(&inst)
		if endptErr != nil {
			sdc.log.Error(endptErr, "skipping instance to endpoint conversion", "instanceId", *inst.InstanceId)
			continue
		}
		endpts = append(endpts, endpt)
	}

	sdc.cache.CacheEndpoints(nsName, svcName, endpts)

	return endpts, nil
}

func (sdc *serviceDiscoveryClient) getNamespace(ctx context.Context, nsName string) (namespace *model.Namespace, err error) {
	// We are assuming a unique namespace name per account
	namespace, exists := sdc.cache.GetNamespace(nsName)
	if exists {
		return namespace, nil
	}

	namespaces, err := sdc.sdApi.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces {
		sdc.cache.CacheNamespace(ns)
		// Set the return namespace
		if nsName == ns.Name {
			namespace = ns
		}
	}

	if namespace == nil {
		// This will cache empty namespace for namespaces not in Cloud Map
		// This is so that we can avoid ListNamespaces call
		sdc.cache.CacheNilNamespace(nsName)
	}

	return namespace, nil
}

func (sdc *serviceDiscoveryClient) getServiceId(ctx context.Context, nsName string, svcName string) (svcId string, err error) {
	svcId, found := sdc.cache.GetServiceId(nsName, svcName)
	if found {
		return svcId, nil
	}

	namespace, err := sdc.getNamespace(ctx, nsName)
	if err != nil || namespace == nil {
		return "", err
	}

	services, err := sdc.sdApi.ListServices(ctx, namespace.Id)
	if err != nil {
		return "", err
	}

	for _, svc := range services {
		sdc.cache.CacheServiceId(nsName, svcName, svc.Id)
		if svc.Name == svcName {
			svcId = svc.Id
		}
	}

	return svcId, nil
}

func (sdc *serviceDiscoveryClient) createNamespace(ctx context.Context, nsName string) (namespace *model.Namespace, err error) {
	sdc.log.Info("creating a new namespace", "namespace", nsName)
	opId, err := sdc.sdApi.CreateHttpNamespace(ctx, nsName)
	if err != nil {
		return nil, err
	}

	nsId, err := sdc.sdApi.PollNamespaceOperation(ctx, opId)
	if err != nil {
		return nil, err
	}

	sdc.log.Info("namespace created", "nsId", nsId)

	// Default namespace type HTTP
	namespace = &model.Namespace{
		Id:   nsId,
		Name: nsName,
		Type: model.HttpNamespaceType,
	}

	sdc.cache.CacheNamespace(namespace)
	return namespace, nil
}
