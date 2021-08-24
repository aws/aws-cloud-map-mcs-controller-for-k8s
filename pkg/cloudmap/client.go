package cloudmap

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-k8s-mcs-controller/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

const (
	defaultNamespaceIdCacheTTL  = 2 * time.Minute
	defaultNamespaceIdCacheSize = 100
	defaultServiceIdCacheTTL    = 2 * time.Minute
	defaultServiceIdCacheSize   = 1024
)

type ServiceDiscoveryClient interface {
	// ListServices returns all services and their endpoints for a given namespace
	ListServices(ctx context.Context, namespaceName string) ([]*model.Service, error)

	// CreateService creates a Cloud Map service resource and return created service struct
	CreateService(ctx context.Context, service *model.Service) error

	// GetService returns a service resource fetched from the Cloud Map API or nil if not found
	GetService(ctx context.Context, namespace string, name string) (*model.Service, error)

	// RegisterEndpoints registers all endpoints for given service
	RegisterEndpoints(ctx context.Context, service *model.Service) error

	// DeleteEndpoints de-registers all endpoints for given service
	DeleteEndpoints(ctx context.Context, service *model.Service) error
}

type serviceDiscoveryClient struct {
	log              logr.Logger
	sdApi            *sd.Client
	namespaceIdCache *cache.LRUExpireCache
	serviceIdCache   *cache.LRUExpireCache
	EndpointManager
}

func NewServiceDiscoveryClient(cfg *aws.Config) ServiceDiscoveryClient {
	return &serviceDiscoveryClient{
		log:              ctrl.Log.WithName("cloudmap"),
		sdApi:            sd.NewFromConfig(*cfg),
		namespaceIdCache: cache.NewLRUExpireCache(defaultNamespaceIdCacheSize),
		serviceIdCache:   cache.NewLRUExpireCache(defaultServiceIdCacheSize),
		EndpointManager:  NewEndpointManager(cfg),
	}
}

func (sdc *serviceDiscoveryClient) ListServices(ctx context.Context, namespaceName string) ([]*model.Service, error) {
	svcs := make([]*model.Service, 0)

	svcSums, svcErr := sdc.listServicesFromCloudMap(ctx, namespaceName)

	if svcErr != nil {
		return svcs, svcErr
	}

	for _, svcSum := range svcSums {
		endpts, endptsErr := sdc.EndpointManager.ListEndpoints(ctx, aws.ToString(svcSum.Id))

		if endptsErr != nil {
			return svcs, endptsErr
		}

		svcs = append(svcs, &model.Service{
			Namespace: namespaceName,
			Name:      aws.ToString(svcSum.Name),
			Endpoints: endpts,
		})
	}

	return svcs, nil
}

func (sdc *serviceDiscoveryClient) CreateService(ctx context.Context, service *model.Service) error {
	sdc.log.Info("creating a new service", "namespace", service.Namespace, "name", service.Name)

	nsId, nsErr := sdc.getNamespaceId(ctx, service.Namespace)

	if nsErr != nil {
		return nsErr
	}

	//TODO: Handle non-http namespaces
	sdSrv, srvErr := sdc.sdApi.CreateService(ctx, &sd.CreateServiceInput{
		Name:        &service.Name,
		NamespaceId: &nsId})

	if srvErr != nil {
		return srvErr
	}

	sdc.serviceIdCache.Add(
		sdc.buildServiceIdCacheKey(nsId, service.Name),
		*sdSrv.Service.Id, defaultServiceIdCacheTTL)

	return sdc.RegisterEndpoints(ctx, service)
}

func (sdc *serviceDiscoveryClient) GetService(ctx context.Context, namespaceName string, serviceName string) (*model.Service, error) {
	sdc.log.Info("fetching a service", "namespaceName", namespaceName, "serviceName", serviceName)

	svcId, svcIdErr := sdc.getServiceId(ctx, namespaceName, serviceName)

	if svcIdErr != nil {
		return nil, svcIdErr
	}

	if svcId == "" {
		return nil, nil
	}

	endpts, endptsErr := sdc.EndpointManager.ListEndpoints(ctx, svcId)

	if endptsErr != nil {
		return nil, endptsErr
	}

	svc := &model.Service{
		Namespace: namespaceName,
		Name:      serviceName,
		Endpoints: endpts,
	}

	return svc, nil
}

func (sdc *serviceDiscoveryClient) RegisterEndpoints(ctx context.Context, service *model.Service) error {
	if len(service.Endpoints) == 0 {
		sdc.log.Info("skipping endpoint registration for empty endpoint list", "serviceName", service.Name)
		return nil
	}
	sdc.log.Info("registering endpoints", "namespaceName", service.Namespace,
		"serviceName", service.Name, "endpoints", service.Endpoints)

	svcId, svcErr := sdc.getServiceId(ctx, service.Namespace, service.Name)
	if svcErr != nil {
		return svcErr
	}

	return sdc.EndpointManager.RegisterEndpoints(ctx, service, svcId)
}

func (sdc *serviceDiscoveryClient) DeleteEndpoints(ctx context.Context, service *model.Service) error {
	if len(service.Endpoints) == 0 {
		sdc.log.Info("skipping endpoint deletion for empty endpoint list", "serviceName", service.Name)
		return nil
	}
	sdc.log.Info("deleting endpoints", "namespaceName", service.Namespace,
		"serviceName", service.Name, "endpoints", service.Endpoints)

	svcId, svcErr := sdc.getServiceId(ctx, service.Namespace, service.Name)
	if svcErr != nil {
		return svcErr
	}

	return sdc.EndpointManager.DeregisterEndpoints(ctx, service, svcId)
}

func (sdc *serviceDiscoveryClient) getNamespaceId(ctx context.Context, nsName string) (string, error) {
	// We are assuming a unique namespace name per account
	if cachedValue, exists := sdc.namespaceIdCache.Get(nsName); exists {
		return cachedValue.(string), nil
	}

	nsId, err := sdc.getNamespaceIdFromCloudMap(ctx, nsName)

	if err != nil {
		return "", err
	}

	sdc.namespaceIdCache.Add(nsName, nsId, defaultNamespaceIdCacheTTL)

	return nsId, err
}

func (sdc *serviceDiscoveryClient) getNamespaceIdFromCloudMap(ctx context.Context, nsName string) (string, error) {

	pages := sd.NewListNamespacesPaginator(sdc.sdApi, &sd.ListNamespacesInput{})

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

	return "", errors.New(fmt.Sprintf("namespace %s not found", nsName))
}

func (sdc *serviceDiscoveryClient) getServiceId(ctx context.Context, nsName string, svcName string) (string, error) {
	cacheKey := sdc.buildServiceIdCacheKey(nsName, svcName)

	if cachedValue, exists := sdc.serviceIdCache.Get(cacheKey); exists {
		return cachedValue.(string), nil
	}

	svcId, svcErr := sdc.getServiceIdFromCloudMap(ctx, nsName, svcName)

	if svcErr != nil {
		return "", svcErr
	}

	if svcId != "" {
		sdc.serviceIdCache.Add(cacheKey, svcId, defaultServiceIdCacheTTL)
	}

	return svcId, nil
}

func (sdc *serviceDiscoveryClient) getServiceIdFromCloudMap(ctx context.Context, nsName string, svcName string) (string, error) {
	svcs, err := sdc.listServicesFromCloudMap(ctx, nsName)

	if err != nil {
		return "", err
	}

	for _, svc := range svcs {
		if svcName == aws.ToString(svc.Name) {
			return aws.ToString(svc.Id), nil
		}
	}

	return "", nil
}

func (sdc *serviceDiscoveryClient) listServicesFromCloudMap(ctx context.Context, nsName string) ([]*types.ServiceSummary, error) {
	svcs := make([]*types.ServiceSummary, 0)

	nsId, nsErr := sdc.getNamespaceId(ctx, nsName)

	if nsErr != nil {
		return svcs, nsErr
	}

	filter := types.ServiceFilter{
		Name:   types.ServiceFilterNameNamespaceId,
		Values: []string{nsId},
	}

	pages := sd.NewListServicesPaginator(sdc.sdApi, &sd.ListServicesInput{Filters: []types.ServiceFilter{filter}})

	for pages.HasMorePages() {
		output, err := pages.NextPage(ctx)
		if err != nil {
			return svcs, err
		}

		for _, svc := range output.Services {
			svcs = append(svcs, &svc)

			cacheKey := sdc.buildServiceIdCacheKey(nsName, aws.ToString(svc.Name))
			svcId := aws.ToString(svc.Id)
			sdc.serviceIdCache.Add(cacheKey, svcId, defaultServiceIdCacheTTL)
		}
	}

	return svcs, nil
}

func (sdc *serviceDiscoveryClient) buildServiceIdCacheKey(nsName string, svcName string) string {
	return fmt.Sprintf("%s/%s", nsName, svcName)
}
