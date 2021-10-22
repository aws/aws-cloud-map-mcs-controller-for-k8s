package test

import "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"

const (
	NsName           = "ns-name"
	NsId             = "ns-id"
	NsHttpType       = "HTTP"
	NsPrivateDnsType = "DNS_PRIVATE"
	SvcName          = "svc-name"
	SvcId            = "svc-id"
	EndptId1         = "endpoint-id-1"
	EndptIp1         = "endpoint-ip-1"
	EndptPort1       = 2
	OpId1            = "operation-id-1"
	OpId2            = "operation-id-2"
	OpStart          = 1
)

func GetTestHttpNamespace() *model.Namespace {
	return &model.Namespace{
		Id:   NsId,
		Name: NsName,
		Type: NsHttpType,
	}
}

func GetTestDnsNamespace() *model.Namespace {
	return &model.Namespace{
		Id:   NsId,
		Name: NsName,
		Type: NsPrivateDnsType,
	}
}

func GetTestService() *model.Service {
	endPt := GetTestEndpoint()
	return &model.Service{
		Namespace: NsName,
		Name:      SvcName,
		Endpoints: []*model.Endpoint{endPt},
	}
}

func GetTestEndpoint() *model.Endpoint {
	return &model.Endpoint{
		Id:         EndptId1,
		IP:         EndptIp1,
		Port:       EndptPort1,
		Attributes: make(map[string]string, 0),
	}
}
