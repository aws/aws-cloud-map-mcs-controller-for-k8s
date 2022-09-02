package model

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
)

var instId = "my-instance"
var ip = "192.168.0.1"
var clusterId = "test-mcs-clusterId"
var clusterSetId = "test-mcs-clusterSetId"
var serviceType = ClusterSetIPType.String()
var svcExportCreationTimestamp int64 = 1640995200000

func TestNewEndpointFromInstance(t *testing.T) {
	tests := []struct {
		name    string
		inst    *types.HttpInstanceSummary
		want    *Endpoint
		wantErr bool
	}{
		{
			name: "happy case",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					ClusterIdAttr:             clusterId,
					ClusterSetIdAttr:          clusterSetId,
					EndpointIpv4Attr:          ip,
					EndpointPortAttr:          "80",
					EndpointProtocolAttr:      "TCP",
					EndpointPortNameAttr:      "http",
					EndpointReadyAttr:         "true",
					ServicePortNameAttr:       "http",
					ServiceProtocolAttr:       "TCP",
					ServicePortAttr:           "65535",
					ServiceTargetPortAttr:     "80",
					ServiceTypeAttr:           serviceType,
					ServiceExportCreationAttr: strconv.FormatInt(svcExportCreationTimestamp, 10),
					"custom-attr":             "custom-val",
				},
			},
			want: &Endpoint{
				Id: instId,
				IP: ip,
				EndpointPort: Port{
					Name:     "http",
					Port:     80,
					Protocol: "TCP",
				},
				ServicePort: Port{
					Name:       "http",
					Port:       65535,
					TargetPort: "80",
					Protocol:   "TCP",
				},
				ClusterId:                      clusterId,
				ClusterSetId:                   clusterSetId,
				ServiceType:                    ServiceType(serviceType),
				ServiceExportCreationTimestamp: svcExportCreationTimestamp,
				Ready:                          true,
				Attributes: map[string]string{
					"custom-attr": "custom-val",
				},
			},
		},
		{
			name: "invalid port",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					EndpointIpv4Attr:      ip,
					EndpointPortAttr:      "80",
					EndpointProtocolAttr:  "TCP",
					EndpointPortNameAttr:  "http",
					EndpointReadyAttr:     "true",
					ServicePortNameAttr:   "http",
					ServiceProtocolAttr:   "TCP",
					ServicePortAttr:       "99999",
					ServiceTargetPortAttr: "80",
					ServiceTypeAttr:       serviceType,
					"custom-attr":         "custom-val",
				},
			},
			wantErr: true,
		},
		{
			name: "missing IP",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					EndpointPortAttr: "80",
					"custom-attr":    "custom-val",
				},
			},
			wantErr: true,
		},
		{
			name: "missing port",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					EndpointIpv4Attr: ip,
					"custom-attr":    "custom-val",
				},
			},
			wantErr: true,
		},
		{
			name: "missing clusterid",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					ClusterSetIdAttr:      clusterSetId,
					EndpointIpv4Attr:      ip,
					EndpointPortAttr:      "80",
					EndpointProtocolAttr:  "TCP",
					EndpointPortNameAttr:  "http",
					EndpointReadyAttr:     "true",
					ServicePortNameAttr:   "http",
					ServiceProtocolAttr:   "TCP",
					ServicePortAttr:       "65535",
					ServiceTargetPortAttr: "80",
					"custom-attr":         "custom-val",
				},
			},
			wantErr: true,
		},
		{
			name: "missing clustersetid",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					ClusterIdAttr:         clusterId,
					EndpointIpv4Attr:      ip,
					EndpointPortAttr:      "80",
					EndpointProtocolAttr:  "TCP",
					EndpointPortNameAttr:  "http",
					EndpointReadyAttr:     "true",
					ServicePortNameAttr:   "http",
					ServiceProtocolAttr:   "TCP",
					ServicePortAttr:       "65535",
					ServiceTargetPortAttr: "80",
					"custom-attr":         "custom-val",
				},
			},
			wantErr: true,
		},
		{
			name: "missing service type",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					ClusterIdAttr:         clusterId,
					ClusterSetIdAttr:      clusterSetId,
					EndpointIpv4Attr:      ip,
					EndpointPortAttr:      "80",
					EndpointProtocolAttr:  "TCP",
					EndpointPortNameAttr:  "http",
					EndpointReadyAttr:     "true",
					ServicePortNameAttr:   "http",
					ServiceProtocolAttr:   "TCP",
					ServicePortAttr:       "65535",
					ServiceTargetPortAttr: "80",
					"custom-attr":         "custom-val",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEndpointFromInstance(tt.inst)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEndpointFromInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEndpointFromInstance() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpoint_GetAttributes(t *testing.T) {
	tests := []struct {
		name     string
		endpoint Endpoint
		want     map[string]string
	}{
		{
			name: "happy case",
			endpoint: Endpoint{
				IP: ip,
				EndpointPort: Port{
					Name:     "http",
					Port:     80,
					Protocol: "TCP",
				},
				ServicePort: Port{
					Name:       "http",
					Port:       30,
					TargetPort: "80",
					Protocol:   "TCP",
				},
				Ready:                          true,
				ClusterId:                      clusterId,
				ClusterSetId:                   clusterSetId,
				ServiceType:                    ServiceType(serviceType),
				ServiceExportCreationTimestamp: svcExportCreationTimestamp,
				Attributes: map[string]string{
					"custom-attr": "custom-val",
				},
			},
			want: map[string]string{
				ClusterIdAttr:             clusterId,
				ClusterSetIdAttr:          clusterSetId,
				EndpointIpv4Attr:          ip,
				EndpointPortAttr:          "80",
				EndpointProtocolAttr:      "TCP",
				EndpointPortNameAttr:      "http",
				EndpointReadyAttr:         "true",
				EndpointHostnameAttr:      "",
				EndpointNodeNameAttr:      "",
				ServicePortNameAttr:       "http",
				ServiceProtocolAttr:       "TCP",
				ServicePortAttr:           "30",
				ServiceTargetPortAttr:     "80",
				ServiceTypeAttr:           serviceType,
				ServiceExportCreationAttr: strconv.FormatInt(svcExportCreationTimestamp, 10),
				"custom-attr":             "custom-val",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.endpoint.GetCloudMapAttributes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpointIdFromIPAddressAndPort(t *testing.T) {
	tests := []struct {
		name    string
		address string
		port    Port
		want    string
	}{
		{
			name:    "happy case",
			address: ip,
			port: Port{
				Name:     "http",
				Port:     80,
				Protocol: "TCP",
			},
			want: "tcp-192_168_0_1-80",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EndpointIdFromIPAddressAndPort(tt.address, tt.port); got != tt.want {
				t.Errorf("EndpointIdFromIPAddressAndPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpoint_Equals(t *testing.T) {
	firstEndpoint := Endpoint{
		Id: instId,
		IP: ip,
		ServicePort: Port{
			Port: 80,
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	secondEndpoint := Endpoint{
		Id: instId,
		IP: ip,
		ServicePort: Port{
			Port: 80,
			Name: "",
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	thirdEndpoint := Endpoint{
		Id: instId,
		IP: ip,
		ServicePort: Port{
			Port: 80,
		},
		Attributes: map[string]string{
			"custom-key": "different-val",
		},
	}

	tests := []struct {
		name string
		x    Endpoint
		y    Endpoint
		want bool
	}{
		{
			name: "identical",
			x:    firstEndpoint,
			y:    secondEndpoint,
			want: true,
		},
		{
			name: "different",
			x:    firstEndpoint,
			y:    thirdEndpoint,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.x.Equals(&tt.y); got != tt.want {
				t.Errorf("Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
