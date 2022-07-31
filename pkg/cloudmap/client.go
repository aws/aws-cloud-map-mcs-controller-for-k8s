package cloudmap

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
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
	log   common.Logger
	sdApi ServiceDiscoveryApi
	cache ServiceDiscoveryClientCache
}

// NewDefaultServiceDiscoveryClient creates a new service discovery client for AWS Cloud Map with default resource cache
// from a given AWS client config.
func NewDefaultServiceDiscoveryClient(cfg *aws.Config) ServiceDiscoveryClient {
	return &serviceDiscoveryClient{
		log:   common.NewLogger("cloudmap"),
		sdApi: NewServiceDiscoveryApiFromConfig(cfg),
		cache: NewDefaultServiceDiscoveryClientCache(),
	}
}

func NewServiceDiscoveryClientWithCustomCache(cfg *aws.Config, cacheConfig *SdCacheConfig) ServiceDiscoveryClient {
	return &serviceDiscoveryClient{
		log:   common.NewLogger("cloudmap"),
		sdApi: NewServiceDiscoveryApiFromConfig(cfg),
		cache: NewServiceDiscoveryClientCache(cacheConfig),
	}
}

func (sdc *serviceDiscoveryClient) ListServices(ctx context.Context, nsName string) (svcs []*model.Service, err error) {
	svcIdMap, err := sdc.getServiceIds(ctx, nsName)
	if err != nil {
		return svcs, err
	}

	for svcName := range svcIdMap {
		endpts, endptsErr := sdc.getEndpoints(ctx, nsName, svcName)
		if endptsErr != nil {
			return svcs, endptsErr
		}

		svcs = append(svcs, &model.Service{
			Namespace: nsName,
			Name:      svcName,
			Endpoints: endpts,
		})
	}

	return svcs, nil
}

func (sdc *serviceDiscoveryClient) CreateService(ctx context.Context, nsName string, svcName string) error {
	sdc.log.Info("creating a new service", "namespace", nsName, "name", svcName)

	nsMap, err := sdc.getNamespaces(ctx)
	if err != nil {
		return err
	}

	namespace := nsMap[nsName]
	if namespace == nil {
		// Create HttpNamespace if the namespace is not present in CloudMap
		namespace, err = sdc.createNamespace(ctx, nsName)
		if err != nil {
			return err
		}
	}

	_, err = sdc.sdApi.CreateService(ctx, *namespace, svcName)
	if err != nil {
		return err
	}

	sdc.cache.EvictServiceIdMap(nsName)

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

	svcIdMap, err := sdc.getServiceIds(ctx, nsName)
	if err != nil {
		return nil, err
	}
	_, found := svcIdMap[svcName]
	if !found {
		return nil, nil
	}

	endpts, err = sdc.getEndpoints(ctx, nsName, svcName)

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

	svcIdMap, err := sdc.getServiceIds(ctx, nsName)
	if err != nil {
		return err
	}
	svcId, found := svcIdMap[svcName]
	if !found {
		return fmt.Errorf("service not found in Cloud Map: %s", svcName)
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
		return errors.New("failure while registering endpoints")
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

	svcIdMap, err := sdc.getServiceIds(ctx, nsName)
	if err != nil {
		return err
	}
	svcId, found := svcIdMap[svcName]
	if !found {
		return fmt.Errorf("service not found in Cloud Map: %s", svcName)
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
		return errors.New("failure while de-registering endpoints")
	}

	return nil
}

func (sdc *serviceDiscoveryClient) getEndpoints(ctx context.Context, nsName string, svcName string) (endpts []*model.Endpoint, err error) {
	endpts, cacheHit := sdc.cache.GetEndpoints(nsName, svcName)
	if cacheHit {
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

func (sdc *serviceDiscoveryClient) getNamespaces(ctx context.Context) (namespace map[string]*model.Namespace, err error) {
	// We are assuming a unique namespace name per account
	namespaces, cacheHit := sdc.cache.GetNamespaceMap()
	if cacheHit {
		return namespaces, nil
	}

	namespaces, err = sdc.sdApi.GetNamespaceMap(ctx)
	if err != nil {
		return nil, err
	}
	sdc.cache.CacheNamespaceMap(namespaces)

	return namespaces, nil
}

func (sdc *serviceDiscoveryClient) getServiceIds(ctx context.Context, nsName string) (map[string]string, error) {
	serviceIdMap, cacheHit := sdc.cache.GetServiceIdMap(nsName)
	if cacheHit {
		return serviceIdMap, nil
	}

	nsMap, err := sdc.getNamespaces(ctx)
	namespace := nsMap[nsName]
	if err != nil || namespace == nil {
		return nil, err
	}

	serviceIdMap, err = sdc.sdApi.GetServiceIdMap(ctx, namespace.Id)
	if err != nil {
		return nil, err
	}
	sdc.cache.CacheServiceIdMap(nsName, serviceIdMap)

	return serviceIdMap, nil
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

	sdc.cache.EvictNamespaceMap()
	return namespace, nil
}
