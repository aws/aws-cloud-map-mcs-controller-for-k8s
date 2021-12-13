package cloudmap

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"k8s.io/apimachinery/pkg/util/cache"
)

const (
	nsKey           = "ns-map"
	svcKeyPrefix    = "svc-map"
	endptsKeyPrefix = "endpts"

	defaultCacheSize = 1024
	defaultNsTTL     = 10 * time.Second
	defaultSvcTTL    = 10 * time.Second
	defaultEndptTTL  = 5 * time.Second
)

type ServiceDiscoveryClientCache interface {
	GetNamespaceMap() (namespaces map[string]*model.Namespace, found bool)
	CacheNamespaceMap(namespaces map[string]*model.Namespace)
	EvictNamespaceMap()
	GetServiceIdMap(namespaceName string) (serviceIdMap map[string]string, found bool)
	CacheServiceIdMap(namespaceName string, serviceIdMap map[string]string)
	EvictServiceIdMap(namespaceName string)
	GetEndpoints(namespaceName string, serviceName string) (endpoints []*model.Endpoint, found bool)
	CacheEndpoints(namespaceName string, serviceName string, endpoints []*model.Endpoint)
	EvictEndpoints(namespaceName string, serviceName string)
}

type sdCache struct {
	log    common.Logger
	cache  *cache.LRUExpireCache
	config *SdCacheConfig
}

type SdCacheConfig struct {
	NsTTL    time.Duration
	SvcTTL   time.Duration
	EndptTTL time.Duration
}

func NewServiceDiscoveryClientCache(cacheConfig *SdCacheConfig) ServiceDiscoveryClientCache {
	return &sdCache{
		log:    common.NewLogger("cloudmap"),
		cache:  cache.NewLRUExpireCache(defaultCacheSize),
		config: cacheConfig,
	}
}

func NewDefaultServiceDiscoveryClientCache() ServiceDiscoveryClientCache {
	return NewServiceDiscoveryClientCache(
		&SdCacheConfig{
			NsTTL:    defaultNsTTL,
			SvcTTL:   defaultSvcTTL,
			EndptTTL: defaultEndptTTL,
		})
}

func (sdCache *sdCache) GetNamespaceMap() (namespaceMap map[string]*model.Namespace, found bool) {
	entry, exists := sdCache.cache.Get(nsKey)
	if !exists {
		return nil, false
	}

	namespaceMap, ok := entry.(map[string]*model.Namespace)
	if !ok {
		sdCache.log.Error(errors.New("failed to retrieve namespaceMap from cache"), "")
		sdCache.cache.Remove(nsKey)
		return nil, false
	}

	return namespaceMap, true
}

func (sdCache *sdCache) CacheNamespaceMap(namespaces map[string]*model.Namespace) {
	sdCache.cache.Add(nsKey, namespaces, sdCache.config.NsTTL)
}

func (sdCache *sdCache) EvictNamespaceMap() {
	sdCache.cache.Remove(nsKey)
}

func (sdCache *sdCache) GetServiceIdMap(nsName string) (serviceIdMap map[string]string, found bool) {
	key := sdCache.buildSvcKey(nsName)
	entry, exists := sdCache.cache.Get(key)
	if !exists {
		return nil, false
	}

	serviceIdMap, ok := entry.(map[string]string)
	if !ok {
		sdCache.log.Error(errors.New("failed to retrieve service IDs from cache"), "",
			"nsName", nsName)
		sdCache.cache.Remove(key)
		return nil, false
	}

	return serviceIdMap, true
}

func (sdCache *sdCache) CacheServiceIdMap(nsName string, serviceIdMap map[string]string) {
	key := sdCache.buildSvcKey(nsName)
	sdCache.cache.Add(key, serviceIdMap, sdCache.config.SvcTTL)
}

func (sdCache *sdCache) EvictServiceIdMap(nsName string) {
	key := sdCache.buildSvcKey(nsName)
	sdCache.cache.Remove(key)
}

func (sdCache *sdCache) GetEndpoints(nsName string, svcName string) (endpts []*model.Endpoint, found bool) {
	key := sdCache.buildEndptsKey(nsName, svcName)
	entry, exists := sdCache.cache.Get(key)
	if !exists {
		return nil, false
	}

	endpts, ok := entry.([]*model.Endpoint)
	if !ok {
		sdCache.log.Error(errors.New("failed to retrieve endpoints from cache"), "",
			"ns", "nsName", "svc", svcName)
		sdCache.cache.Remove(key)
		return nil, false
	}

	return endpts, true
}

func (sdCache *sdCache) CacheEndpoints(nsName string, svcName string, endpts []*model.Endpoint) {
	key := sdCache.buildEndptsKey(nsName, svcName)
	sdCache.cache.Add(key, endpts, sdCache.config.EndptTTL)
}

func (sdCache *sdCache) EvictEndpoints(nsName string, svcName string) {
	key := sdCache.buildEndptsKey(nsName, svcName)
	sdCache.cache.Remove(key)
}

func (sdCache *sdCache) buildSvcKey(nsName string) (cacheKey string) {
	return fmt.Sprintf("%s:%s", svcKeyPrefix, nsName)
}

func (sdCache *sdCache) buildEndptsKey(nsName string, svcName string) string {
	return fmt.Sprintf("%s:%s:%s", endptsKeyPrefix, nsName, svcName)
}
