package controllers

import (
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
)

func ServicePortToPort(svcPort v1.ServicePort) model.Port {
	return model.Port{
		Name:       svcPort.Name,
		Port:       svcPort.Port,
		TargetPort: svcPort.TargetPort.String(),
		Protocol:   protocolToString(svcPort.Protocol),
	}
}

func EndpointPortToPort(port discovery.EndpointPort) model.Port {
	return model.Port{
		Name:     *port.Name,
		Port:     *port.Port,
		Protocol: protocolToString(*port.Protocol),
	}
}

func PortToServicePort(port model.Port) v1.ServicePort {
	return v1.ServicePort{
		Name:       port.Name,
		Protocol:   stringToProtocol(port.Protocol),
		Port:       port.Port,
		TargetPort: intstr.Parse(port.TargetPort),
	}
}

func PortToEndpointPort(port model.Port) discovery.EndpointPort {
	protocol := stringToProtocol(port.Protocol)
	return discovery.EndpointPort{
		Name:     &port.Name,
		Protocol: &protocol,
		Port:     &port.Port,
	}
}

func protocolToString(protocol v1.Protocol) string {
	switch protocol {
	case v1.ProtocolTCP:
		return model.TCPProtocol
	case v1.ProtocolUDP:
		return model.UDPProtocol
	case v1.ProtocolSCTP:
		return model.SCTPProtocol
	default:
		return ""
	}
}

func stringToProtocol(protocol string) v1.Protocol {
	switch protocol {
	case model.TCPProtocol:
		return v1.ProtocolTCP
	case model.UDPProtocol:
		return v1.ProtocolUDP
	case model.SCTPProtocol:
		return v1.ProtocolSCTP
	default:
		return ""
	}
}

func EndpointPortsAreEqualIgnoreOrder(a, b []discovery.EndpointPort) (equal bool) {
	if len(a) != len(b) {
		return false
	}

	for _, aPort := range a {
		match := false
		for _, bPort := range b {
			if reflect.DeepEqual(aPort, bPort) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}
