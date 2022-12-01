package cloudmap

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
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
	log          common.Logger
	sdApi        ServiceDiscoveryApi
	cache        ServiceDiscoveryClientCache
	clusterUtils model.ClusterUtils
}

// NewDefaultServiceDiscoveryClient creates a new service discovery client for AWS Cloud Map with default resource cache
// from a given AWS client config.
func NewDefaultServiceDiscoveryClient(cfg *aws.Config, clusterUtils model.ClusterUtils) ServiceDiscoveryClient {
	return &serviceDiscoveryClient{
		log:          common.NewLogger("cloudmap", "client"),
		sdApi:        NewServiceDiscoveryApiFromConfig(cfg),
		cache:        NewDefaultServiceDiscoveryClientCache(),
		clusterUtils: clusterUtils,
	}
}

func NewServiceDiscoveryClientWithCustomCache(cfg *aws.Config, cacheConfig *SdCacheConfig, clusterUtils model.ClusterUtils) ServiceDiscoveryClient {
	return &serviceDiscoveryClient{
		log:          common.NewLogger("cloudmap", "client"),
		sdApi:        NewServiceDiscoveryApiFromConfig(cfg),
		cache:        NewServiceDiscoveryClientCache(cacheConfig),
		clusterUtils: clusterUtils,
	}
}

func (sdc *serviceDiscoveryClient) ListServices(ctx context.Context, nsName string) (svcs []*model.Service, err error) {
	svcIdMap, err := sdc.getServiceIds(ctx, nsName)
	if err != nil {
		// Ignore resource not found error, as it will indicate deleted resources in CloudMap
		if common.IsNotFound(err) {
			return svcs, nil
		}
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

	namespace, err := sdc.getNamespace(ctx, nsName)
	if common.IsUnknown(err) {
		return err
	}

	if common.IsNotFound(err) {
		sdc.log.Info("namespace not found for service", "namespace", nsName, "service", svcName)
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

	_, err = sdc.getServiceId(ctx, nsName, svcName)
	if err != nil {
		return nil, err
	}

	endpts, err := sdc.getEndpointsInThisCluster(ctx, nsName, svcName, true)
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

	operationPoller := NewOperationPoller(sdc.sdApi)
	for _, endpt := range endpts {
		endptId := endpt.Id
		endptAttrs := endpt.GetCloudMapAttributes()
		operationPoller.Submit(ctx, func() (opId string, err error) {
			return sdc.sdApi.RegisterInstance(ctx, svcId, endptId, endptAttrs)
		})
	}

	// Evict cache entry so next list call reflects changes
	sdc.cache.EvictEndpoints(nsName, svcName)

	err = operationPoller.Await()
	if err != nil {
		return common.Wrap(err, errors.New("failure while registering endpoints"))
	}

	return nil
}

func (sdc *serviceDiscoveryClient) DeleteEndpoints(ctx context.Context, nsName string, svcName string, endpts []*model.Endpoint) (err error) {
	if len(endpts) == 0 {
		sdc.log.Info("skipping endpoint deletion for empty endpoint list", "serviceName", svcName)
		return nil
	}

	sdc.log.Info("deleting endpoints", "namespaceName", nsName, "serviceName", svcName, "endpoints", endpts)

	svcId, err := sdc.getServiceId(ctx, nsName, svcName)
	if err != nil {
		return err
	}

	operationPoller := NewOperationPoller(sdc.sdApi)
	for _, endpt := range endpts {
		endptId := endpt.Id
		operationPoller.Submit(ctx, func() (opId string, err error) {
			return sdc.sdApi.DeregisterInstance(ctx, svcId, endptId)
		})
	}

	// Evict cache entry so next list call reflects changes
	sdc.cache.EvictEndpoints(nsName, svcName)

	err = operationPoller.Await()
	if err != nil {
		return common.Wrap(err, errors.New("failure while de-registering endpoints"))
	}

	return err
}

func (sdc *serviceDiscoveryClient) getEndpoints(ctx context.Context, nsName string, svcName string) (endpts []*model.Endpoint, err error) {
	endpts, found := sdc.cache.GetEndpoints(nsName, svcName)
	if found {
		return endpts, nil
	}

	endpts, err = sdc.getEndpointsInThisCluster(ctx, nsName, svcName, false)
	if err != nil {
		sdc.log.Error(err, "failed to retrieve clusterSetId")
		return nil, err
	}
	sdc.cache.CacheEndpoints(nsName, svcName, endpts)

	return endpts, nil
}

func (sdc *serviceDiscoveryClient) getEndpointsInThisCluster(ctx context.Context, nsName string, svcName string, inThisCluster bool) (endpts []*model.Endpoint, err error) {
	clusterProperties, err := sdc.clusterUtils.GetClusterProperties(ctx)
	if err != nil {
		sdc.log.Error(err, "failed to retrieve clusterSetId")
		return nil, err
	}

	queryParameters := map[string]string{
		model.ClusterSetIdAttr: clusterProperties.ClusterSetId(),
	}
	if inThisCluster {
		queryParameters[model.ClusterIdAttr] = clusterProperties.ClusterId()
	}
	insts, err := sdc.sdApi.DiscoverInstances(ctx, nsName, svcName, queryParameters)
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

	return endpts, nil
}

func (sdc *serviceDiscoveryClient) getNamespace(ctx context.Context, nsName string) (namespace *model.Namespace, err error) {
	namespaces, err := sdc.getNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	if namespace, ok := namespaces[nsName]; ok {
		return namespace, nil
	}

	return nil, common.NotFoundError(fmt.Sprintf("namespace: %s", nsName))
}

func (sdc *serviceDiscoveryClient) getNamespaces(ctx context.Context) (namespaces map[string]*model.Namespace, err error) {
	// We are assuming a unique namespace name per account
	namespaces, found := sdc.cache.GetNamespaceMap()
	if found {
		return namespaces, nil
	}

	namespaces, err = sdc.sdApi.GetNamespaceMap(ctx)
	if err != nil {
		return nil, err
	}
	sdc.cache.CacheNamespaceMap(namespaces)

	return namespaces, nil
}

func (sdc *serviceDiscoveryClient) getServiceId(ctx context.Context, nsName string, svcName string) (svcId string, err error) {
	svcIdMap, err := sdc.getServiceIds(ctx, nsName)
	if err != nil {
		return "", err
	}

	if svcId, ok := svcIdMap[svcName]; ok {
		return svcId, nil
	}

	return "", common.NotFoundError(fmt.Sprintf("service: %s", svcName))
}

func (sdc *serviceDiscoveryClient) getServiceIds(ctx context.Context, nsName string) (map[string]string, error) {
	serviceIdMap, found := sdc.cache.GetServiceIdMap(nsName)
	if found {
		return serviceIdMap, nil
	}

	namespace, err := sdc.getNamespace(ctx, nsName)
	if err != nil {
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

	op, err := NewOperationPoller(sdc.sdApi).Poll(ctx, opId)
	if err != nil {
		return nil, err
	}
	nsId := op.Targets[string(types.OperationTargetTypeNamespace)]

	sdc.log.Info("namespace created", "nsId", nsId, "namespace", nsName)

	// Default namespace type HTTP
	namespace = &model.Namespace{
		Id:   nsId,
		Name: nsName,
		Type: model.HttpNamespaceType,
	}

	sdc.cache.EvictNamespaceMap()
	return namespace, nil
}
