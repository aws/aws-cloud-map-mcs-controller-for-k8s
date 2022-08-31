package controllers

import (
	"strconv"
	"time"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"

	multiclusterv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/aws/aws-sdk-go-v2/aws"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
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
	creationTimestamp := metav1.NewTime(time.UnixMilli(test.SvcExportCreationTimestamp))
	return &multiclusterv1alpha1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:              test.SvcName,
			Namespace:         test.HttpNsName,
			CreationTimestamp: creationTimestamp,
		},
	}
}

func endpointSliceForTest() *discovery.EndpointSlice {
	port := int32(test.Port1)
	protocol := v1.ProtocolTCP
	nodename := test.Nodename
	hostname := test.Hostname
	ready, _ := strconv.ParseBool(test.EndptReadyTrue)
	return &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.HttpNsName,
			Name:      test.SvcName + "-slice",
			Labels:    map[string]string{discovery.LabelServiceName: test.SvcName},
		},
		AddressType: discovery.AddressTypeIPv4,
		Endpoints: []discovery.Endpoint{{
			Addresses: []string{test.EndptIp1},
			Conditions: discovery.EndpointConditions{
				Ready: aws.Bool(ready),
			},
			NodeName: &nodename,
			Hostname: &hostname,
		}},
		Ports: []discovery.EndpointPort{{
			Name:     aws.String(test.PortName1),
			Protocol: &protocol,
			Port:     &port,
		}},
	}
}

func endpointSliceFromEndpointsForTest(endpts []*model.Endpoint, ports []discovery.EndpointPort) *discovery.EndpointSlice {
	svc := k8sServiceForTest()
	slice := CreateEndpointSliceStruct(svc, test.SvcName, test.ClusterId1)
	slice.Ports = ports

	testEndpoints := make([]discovery.Endpoint, 0)
	for _, endpt := range endpts {
		testEndpoints = append(testEndpoints, CreateEndpointForSlice(svc, endpt))
	}
	slice.Endpoints = testEndpoints

	return slice
}
