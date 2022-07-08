package controllers

import (
	multiclusterv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Factory functions for testing

func k8sNamespaceForTest() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.HttpNsName,
			Namespace: test.HttpNsName,
		},
	}
}

func k8sServiceForTest() *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.SvcName,
			Namespace: test.HttpNsName,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       test.PortName1,
				Protocol:   test.Protocol1,
				Port:       test.ServicePort1,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: test.Port1},
			}},
		},
		Status: v1.ServiceStatus{},
	}
}

func serviceExportForTest() *multiclusterv1alpha1.ServiceExport {
	return &multiclusterv1alpha1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test.SvcName,
			Namespace: test.HttpNsName,
		},
	}
}

func endpointSliceForTest() *discovery.EndpointSlice {
	port := int32(test.Port1)
	protocol := v1.ProtocolTCP
	return &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.HttpNsName,
			Name:      test.SvcName + "-slice",
			Labels:    map[string]string{discovery.LabelServiceName: test.SvcName},
		},
		AddressType: discovery.AddressTypeIPv4,
		Endpoints: []discovery.Endpoint{{
			Addresses: []string{test.EndptIp1},
		}},
		Ports: []discovery.EndpointPort{{
			Name:     aws.String(test.PortName1),
			Protocol: &protocol,
			Port:     &port,
		}},
	}
}

func endpointSliceWithIpsAndPortsForTest(ips []string, ports []discovery.EndpointPort) *discovery.EndpointSlice {
	svc := k8sServiceForTest()
	slice := CreateEndpointSliceStruct(svc, test.SvcName)
	slice.Ports = ports

	testEndpoints := make([]discovery.Endpoint, 0)
	for _, ip := range ips {
		testEndpoints = append(testEndpoints, CreateEndpointForSlice(svc, ip))
	}
	slice.Endpoints = testEndpoints

	return slice
}
