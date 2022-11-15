package cloudmap

import (
	"testing"
	"time"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/cache"
)

func TestNewServiceDiscoveryClientCache(t *testing.T) {
	sdc, ok := NewServiceDiscoveryClientCache(&SdCacheConfig{
		NsTTL:    3 * time.Second,
		SvcTTL:   3 * time.Second,
		EndptTTL: 3 * time.Second,
	}).(*sdCache)
	if !ok {
		t.Fatalf("failed to create cache")
	}

	assert.Equal(t, 3*time.Second, sdc.config.NsTTL)
	assert.Equal(t, 3*time.Second, sdc.config.SvcTTL)
	assert.Equal(t, 3*time.Second, sdc.config.EndptTTL)
}

func TestNewDefaultServiceDiscoveryClientCache(t *testing.T) {
	sdc, ok := NewDefaultServiceDiscoveryClientCache().(*sdCache)
	if !ok {
		t.Fatalf("failed to create cache")
	}

	assert.Equal(t, defaultNsTTL, sdc.config.NsTTL)
	assert.Equal(t, defaultSvcTTL, sdc.config.SvcTTL)
	assert.Equal(t, defaultEndptTTL, sdc.config.EndptTTL)
}

func TestServiceDiscoveryClientCacheGetNamespaceMap_Found(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheNamespaceMap(map[string]*model.Namespace{
		test.HttpNsName: test.GetTestHttpNamespace(),
	})

	nsMap, found := sdc.GetNamespaceMap()
	assert.True(t, found)
	assert.Equal(t, test.GetTestHttpNamespace(), nsMap[test.HttpNsName])
}

func TestServiceDiscoveryClientCacheGetNamespaceMap_NotFound(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()

	nsMap, found := sdc.GetNamespaceMap()
	assert.False(t, found)
	assert.Nil(t, nsMap)
}

func TestServiceDiscoveryClientCacheGetNamespaceMap_Corrupt(t *testing.T) {
	sdc := getCacheImpl(t)
	sdc.defaultCache.Add(nsKey, &model.Plan{}, time.Minute)

	nsMap, found := sdc.GetNamespaceMap()
	assert.False(t, found)
	assert.Nil(t, nsMap)
}

func TestServiceDiscoveryClientEvictNamespaceMap(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheNamespaceMap(map[string]*model.Namespace{
		test.HttpNsName: test.GetTestHttpNamespace(),
	})
	sdc.EvictNamespaceMap()

	nsMap, found := sdc.GetNamespaceMap()
	assert.False(t, found)
	assert.Nil(t, nsMap)
}

func TestServiceDiscoveryClientCacheGetServiceIdMap_Found(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheServiceIdMap(test.HttpNsName, map[string]string{
		test.SvcName: test.SvcId,
	})

	svcIdMap, found := sdc.GetServiceIdMap(test.HttpNsName)
	assert.True(t, found)
	assert.Equal(t, test.SvcId, svcIdMap[test.SvcName])
}

func TestServiceDiscoveryClientCacheGetServiceIdMap_NotFound(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()

	svcIdMap, found := sdc.GetServiceIdMap(test.HttpNsName)
	assert.False(t, found)
	assert.Empty(t, svcIdMap)
}

func TestServiceDiscoveryClientCacheGetServiceIdMap_Corrupt(t *testing.T) {
	sdc := getCacheImpl(t)
	sdc.defaultCache.Add(sdc.buildSvcKey(test.HttpNsName), &model.Plan{}, time.Minute)

	svcIdMap, found := sdc.GetServiceIdMap(test.HttpNsName)
	assert.False(t, found)
	assert.Empty(t, svcIdMap)
}

func TestServiceDiscoveryClientEvictServiceIdMap(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheServiceIdMap(test.HttpNsName, map[string]string{
		test.SvcName: test.SvcId,
	})
	sdc.EvictServiceIdMap(test.HttpNsName)

	svcIdMap, found := sdc.GetServiceIdMap(test.HttpNsName)
	assert.False(t, found)
	assert.Empty(t, svcIdMap)
}

func TestServiceDiscoveryClientCacheGetEndpoints_Found(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheEndpoints(test.HttpNsName, test.SvcName, []*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})

	endpts, found := sdc.GetEndpoints(test.HttpNsName, test.SvcName)
	assert.True(t, found)
	assert.Equal(t, []*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()}, endpts)
}

func TestServiceDiscoveryClientCacheGetEndpoints_NotFound(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()

	endpts, found := sdc.GetEndpoints(test.HttpNsName, test.SvcName)
	assert.False(t, found)
	assert.Nil(t, endpts)
}

func TestServiceDiscoveryClientCacheGetEndpoints_Corrupt(t *testing.T) {
	sdc := getCacheImpl(t)
	sdc.defaultCache.Add(sdc.buildEndptsKey(test.HttpNsName, test.SvcName), &model.Plan{}, time.Minute)

	endpts, found := sdc.GetEndpoints(test.HttpNsName, test.SvcName)
	assert.False(t, found)
	assert.Nil(t, endpts)
}

func TestServiceDiscoveryClientEvictEndpoints(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheEndpoints(test.HttpNsName, test.SvcName, []*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})
	sdc.EvictEndpoints(test.HttpNsName, test.SvcName)

	endpts, found := sdc.GetEndpoints(test.HttpNsName, test.SvcName)
	assert.False(t, found)
	assert.Nil(t, endpts)
}

func getCacheImpl(t *testing.T) sdCache {
	return sdCache{
		log:            common.NewLoggerWithLogr(testr.New(t)),
		defaultCache:   cache.NewLRUExpireCache(defaultCacheSize),
		endpointsCache: cache.NewLRUExpireCache(defaultCacheSize),
	}
}
