package cloudmap

import (
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewServiceDiscoveryClientCache(t *testing.T) {
	sdc := NewServiceDiscoveryClientCache(&SdCacheConfig{
		NsTTL:    3 * time.Second,
		SvcTTL:   3 * time.Second,
		EndptTTL: 3 * time.Second,
	}).(*sdCache)

	assert.Equal(t, 3*time.Second, sdc.config.NsTTL)
	assert.Equal(t, 3*time.Second, sdc.config.SvcTTL)
	assert.Equal(t, 3*time.Second, sdc.config.EndptTTL)
}

func TestNewDefaultServiceDiscoveryClientCache(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache().(*sdCache)

	assert.Equal(t, defaultNsTTL, sdc.config.NsTTL)
	assert.Equal(t, defaultSvcTTL, sdc.config.SvcTTL)
	assert.Equal(t, defaultEndptTTL, sdc.config.EndptTTL)
}

func TestServiceDiscoveryClientCacheGetNamespace_Found(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheNamespace(test.GetTestHttpNamespace())

	ns, found := sdc.GetNamespace(test.NsName)
	assert.True(t, found)
	assert.Equal(t, test.GetTestHttpNamespace(), ns)
}

func TestServiceDiscoveryClientCacheGetNamespace_NotFound(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()

	ns, found := sdc.GetNamespace(test.NsName)
	assert.False(t, found)
	assert.Nil(t, ns)
}

func TestServiceDiscoveryClientCacheGetNamespace_Nil(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheNilNamespace(test.NsName)

	ns, found := sdc.GetNamespace(test.NsName)
	assert.True(t, found)
	assert.Nil(t, ns)
}

func TestServiceDiscoveryClientCacheGetNamespace_Corrupt(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache().(*sdCache)
	sdc.cache.Add(sdc.buildNsKey(test.NsName), &model.Resource{}, time.Minute)

	ns, found := sdc.GetNamespace(test.NsName)
	assert.False(t, found)
	assert.Nil(t, ns)
}

func TestServiceDiscoveryClientCacheGetServiceId_Found(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheServiceId(test.NsName, test.SvcName, test.SvcId)

	svcId, found := sdc.GetServiceId(test.NsName, test.SvcName)
	assert.True(t, found)
	assert.Equal(t, test.SvcId, svcId)
}

func TestServiceDiscoveryClientCacheGetServiceId_NotFound(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()

	svcId, found := sdc.GetServiceId(test.NsName, test.SvcName)
	assert.False(t, found)
	assert.Empty(t, svcId)
}

func TestServiceDiscoveryClientCacheGetServiceId_Corrupt(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache().(*sdCache)

	sdc.cache.Add(sdc.buildSvcKey(test.NsName, test.SvcName), &model.Resource{}, time.Minute)
	svcId, found := sdc.GetServiceId(test.NsName, test.SvcName)
	assert.False(t, found)
	assert.Empty(t, svcId)
}

func TestServiceDiscoveryClientCacheGetEndpoints_Found(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheEndpoints(test.NsName, test.SvcName, []*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})

	endpts, found := sdc.GetEndpoints(test.NsName, test.SvcName)
	assert.True(t, found)
	assert.Equal(t, []*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()}, endpts)
}

func TestServiceDiscoveryClientCacheGetEndpoints_NotFound(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()

	endpts, found := sdc.GetEndpoints(test.NsName, test.SvcName)
	assert.False(t, found)
	assert.Nil(t, endpts)
}

func TestServiceDiscoveryClientCacheGetEndpoints_Corrupt(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache().(*sdCache)

	sdc.cache.Add(sdc.buildEndptsKey(test.NsName, test.SvcName), &model.Resource{}, time.Minute)
	endpts, found := sdc.GetEndpoints(test.NsName, test.SvcName)
	assert.False(t, found)
	assert.Nil(t, endpts)
}

func TestServiceDiscoveryClientEvictEndpoints(t *testing.T) {
	sdc := NewDefaultServiceDiscoveryClientCache()
	sdc.CacheEndpoints(test.NsName, test.SvcName, []*model.Endpoint{test.GetTestEndpoint1(), test.GetTestEndpoint2()})
	sdc.EvictEndpoints(test.NsName, test.SvcName)

	endpts, found := sdc.GetEndpoints(test.NsName, test.SvcName)
	assert.False(t, found)
	assert.Nil(t, endpts)
}
