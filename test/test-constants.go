package test

import (
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
)

const (
	NsName          = "ns-name"
	NsId            = "ns-id"
	SvcName         = "svc-name"
	SvcId           = "svc-id"
	EndptId1        = "tcp://192.168.0.1:1"
	EndptId2        = "tcp://192.168.0.2:2"
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
	PortName2       = "http"
	Protocol2       = "TCP"
	ServicePort2    = 22
	ServicePortStr2 = "22"
	OpId1           = "operation-id-1"
	OpId2           = "operation-id-2"
	OpStart         = 1
)

func GetTestHttpNamespace() *model.Namespace {
	return &model.Namespace{
		Id:   NsId,
		Name: NsName,
		Type: model.HttpNamespaceType,
	}
}

func GetTestDnsNamespace() *model.Namespace {
	return &model.Namespace{
		Id:   NsId,
		Name: NsName,
		Type: model.DnsPrivateNamespaceType,
	}
}

func GetTestService() *model.Service {
	return &model.Service{
		Namespace: NsName,
		Name:      SvcName,
		Endpoints: []*model.Endpoint{GetTestEndpoint1(), GetTestEndpoint2()},
	}
}

func GetTestServiceWithEndpoint(endpoints []*model.Endpoint) *model.Service {
	return &model.Service{
		Namespace: NsName,
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
		Attributes: make(map[string]string),
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
		Attributes: make(map[string]string),
	}
}
