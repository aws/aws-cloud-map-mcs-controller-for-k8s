package controllers

import (
	"reflect"
	"testing"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestServicePortToPort(t *testing.T) {
	type args struct {
		svcPort v1.ServicePort
	}
	tests := []struct {
		name string
		args args
		want model.Port
	}{
		{
			name: "happy case",
			args: args{
				svcPort: v1.ServicePort{
					Name:     "http",
					Protocol: v1.ProtocolTCP,
					Port:     80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
					},
				},
			},
			want: model.Port{
				Name:       "http",
				Port:       80,
				TargetPort: "8080",
				Protocol:   "TCP",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServicePortToPort(tt.args.svcPort); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServicePortToPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpointPortToPort(t *testing.T) {
	type args struct {
		port v1beta1.EndpointPort
	}
	name := "http"
	protocolTCP := v1.ProtocolTCP
	port := int32(80)
	tests := []struct {
		name string
		args args
		want model.Port
	}{
		{
			name: "happy case",
			args: args{
				port: v1beta1.EndpointPort{
					Name:     &name,
					Protocol: &protocolTCP,
					Port:     &port,
				},
			},
			want: model.Port{
				Name:       "http",
				Port:       80,
				TargetPort: "",
				Protocol:   "TCP",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EndpointPortToPort(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EndpointPortToPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortToServicePort(t *testing.T) {
	type args struct {
		port model.Port
	}
	tests := []struct {
		name string
		args args
		want v1.ServicePort
	}{
		{
			name: "happy case",
			args: args{
				port: model.Port{
					Name:       "http",
					Port:       80,
					TargetPort: "8080",
					Protocol:   "TCP",
				},
			},
			want: v1.ServicePort{
				Name:     "http",
				Protocol: v1.ProtocolTCP,
				Port:     80,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8080,
				},
			},
		},
		{
			name: "happy case for string targertPort",
			args: args{
				port: model.Port{
					Name:       "http",
					Port:       80,
					TargetPort: "https",
					Protocol:   "TCP",
				},
			},
			want: v1.ServicePort{
				Name:     "http",
				Protocol: v1.ProtocolTCP,
				Port:     80,
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "https",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PortToServicePort(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PortToServicePort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortToEndpointPort(t *testing.T) {
	name := "http"
	protocolTCP := v1.ProtocolTCP
	port := int32(80)
	type args struct {
		port model.Port
	}
	tests := []struct {
		name string
		args args
		want v1beta1.EndpointPort
	}{
		{
			name: "happy case",
			args: args{
				port: model.Port{
					Name:     "http",
					Port:     80,
					Protocol: "TCP",
				},
			},
			want: v1beta1.EndpointPort{
				Name:     &name,
				Protocol: &protocolTCP,
				Port:     &port,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PortToEndpointPort(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PortToEndpointPort() = %v, want %v", got, tt.want)
			}
		})
	}
}
