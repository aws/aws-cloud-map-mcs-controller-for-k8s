package test

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aboutv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/about/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
)

const (
	HttpNsName      = "http-ns-name"
	DnsNsName       = "dns-ns-name"
	HttpNsId        = "http-ns-id"
	DnsNsId         = "dns-ns-id"
	SvcName         = "svc-name"
	SvcId           = "svc-id"
	ClusterId       = "test-mcs-clusterId"
	ClustersetId    = "test-mcs-clusterSetId"
	EndptId1        = "tcp-192_168_0_1-1"
	EndptId2        = "tcp-192_168_0_2-2"
	EndptIp1        = "192.168.0.1"
	EndptIp2        = "192.168.0.2"
	Port1           = 1
	PortStr1        = "1"
	PortName1       = "http"
	Protocol1       = "TCP"
	ServicePort1    = 11
	ServicePortStr1 = "11"
	Port2           = 2
	PortStr2        = "2"
	PortName2       = "https"
	Protocol2       = "UDP"
	ServicePort2    = 22
	ServicePortStr2 = "22"
	OpId1           = "operation-id-1"
	OpId2           = "operation-id-2"
	OpStart         = 1
)

func GetTestHttpNamespace() *model.Namespace {
	return &model.Namespace{
		Id:   HttpNsId,
		Name: HttpNsName,
		Type: model.HttpNamespaceType,
	}
}

func GetTestDnsNamespace() *model.Namespace {
	return &model.Namespace{
		Id:   DnsNsId,
		Name: DnsNsName,
		Type: model.DnsPrivateNamespaceType,
	}
}

func GetTestService() *model.Service {
	return &model.Service{
		Namespace: HttpNsName,
		Name:      SvcName,
		Endpoints: []*model.Endpoint{GetTestEndpoint1(), GetTestEndpoint2()},
	}
}

func GetTestServiceWithEndpoint(endpoints []*model.Endpoint) *model.Service {
	return &model.Service{
		Namespace: HttpNsName,
		Name:      SvcName,
		Endpoints: endpoints,
	}
}

func GetTestEndpoint1() *model.Endpoint {
	return &model.Endpoint{
		Id: EndptId1,
		IP: EndptIp1,
		EndpointPort: model.Port{
			Name:     PortName1,
			Port:     Port1,
			Protocol: Protocol1,
		},
		ServicePort: model.Port{
			Name:       PortName1,
			Port:       ServicePort1,
			TargetPort: PortStr1,
			Protocol:   Protocol1,
		},
		ClusterId:    ClusterId,
		ClusterSetId: ClustersetId,
		Attributes:   make(map[string]string),
	}
}

func GetTestEndpoint2() *model.Endpoint {
	return &model.Endpoint{
		Id: EndptId2,
		IP: EndptIp2,
		EndpointPort: model.Port{
			Name:     PortName2,
			Port:     Port2,
			Protocol: Protocol2,
		},
		ServicePort: model.Port{
			Name:       PortName2,
			Port:       ServicePort2,
			TargetPort: PortStr2,
			Protocol:   Protocol2,
		},
		ClusterId:    ClusterId,
		ClusterSetId: ClustersetId,
		Attributes:   make(map[string]string),
	}
}

func GetTestEndpoints(count int) (endpts []*model.Endpoint) {
	// use +3 offset go avoid collision with test endpoint 1 and 2
	for i := 3; i < count+3; i++ {
		e := GetTestEndpoint1()
		e.ClusterId = ClusterId
		e.Id = fmt.Sprintf("tcp-192_168_0_%d-1", i)
		e.IP = fmt.Sprintf("192.168.0.%d", i)
		endpts = append(endpts, e)
	}
	return endpts
}

func ClusterIdForTest() *aboutv1alpha1.ClusterProperty {
	return &aboutv1alpha1.ClusterProperty{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.ClusterIdName,
		},
		Spec: aboutv1alpha1.ClusterPropertySpec{
			Value: ClusterId,
		},
	}
}

func ClusterSetIdForTest() *aboutv1alpha1.ClusterProperty {
	return &aboutv1alpha1.ClusterProperty{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.ClusterSetIdName,
		},
		Spec: aboutv1alpha1.ClusterPropertySpec{
			Value: ClustersetId,
		},
	}
}
