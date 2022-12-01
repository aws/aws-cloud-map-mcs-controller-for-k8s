package model

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
)

var instId = "my-instance"
var ipv4 = "192.168.0.1"
var ipv6 = "2001:0db8:0001:0000:0000:0ab9:C0A8:0102"
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
			name: "happy case ipv4",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					ClusterIdAttr:             clusterId,
					ClusterSetIdAttr:          clusterSetId,
					EndpointIpv4Attr:          ipv4,
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
				Id:     instId,
				IP:     ipv4,
				IPType: IPV4Type,
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
			name: "happy case ipv6",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					ClusterIdAttr:             clusterId,
					ClusterSetIdAttr:          clusterSetId,
					EndpointIpv6Attr:          ipv6,
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
				Id:     instId,
				IP:     ipv6,
				IPType: IPV6Type,
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
					EndpointIpv4Attr:      ipv4,
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
					EndpointIpv4Attr: ipv4,
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
					EndpointIpv4Attr:      ipv4,
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
					EndpointIpv4Attr:      ipv4,
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
					EndpointIpv4Attr:      ipv4,
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
			name: "happy case ipv4",
			endpoint: Endpoint{
				IP:     ipv4,
				IPType: IPV4Type,
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
				EndpointIpv4Attr:          ipv4,
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
		{
			name: "happy case ipv6",
			endpoint: Endpoint{
				IP:     ipv6,
				IPType: IPV6Type,
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
				EndpointIpv6Attr:          ipv6,
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
			name:    "happy case ipv4",
			address: ipv4,
			port: Port{
				Name:     "http",
				Port:     80,
				Protocol: "TCP",
			},
			want: "tcp-192_168_0_1-80",
		},
		{
			name:    "happy case ipv6",
			address: ipv6,
			port: Port{
				Name:     "http",
				Port:     80,
				Protocol: "TCP",
			},
			want: "tcp-2001_0db8_0001_0000_0000_0ab9_C0A8_0102-80",
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
	firstEndpointIpv4 := Endpoint{
		Id:     instId,
		IP:     ipv4,
		IPType: IPV4Type,
		ServicePort: Port{
			Port: 80,
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	secondEndpointIpv4 := Endpoint{
		Id:     instId,
		IP:     ipv4,
		IPType: IPV4Type,
		ServicePort: Port{
			Port: 80,
			Name: "",
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	thirdEndpointIpv4 := Endpoint{
		Id:     instId,
		IP:     ipv4,
		IPType: IPV4Type,
		ServicePort: Port{
			Port: 80,
		},
		Attributes: map[string]string{
			"custom-key": "different-val",
		},
	}

	firstEndpointIpv6 := Endpoint{
		Id:     instId,
		IP:     ipv6,
		IPType: IPV6Type,
		ServicePort: Port{
			Port: 80,
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	secondEndpointIpv6 := Endpoint{
		Id:     instId,
		IP:     ipv6,
		IPType: IPV6Type,
		ServicePort: Port{
			Port: 80,
			Name: "",
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	thirdEndpointIpv6 := Endpoint{
		Id:     instId,
		IP:     ipv6,
		IPType: IPV6Type,
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
			name: "identical ipv4",
			x:    firstEndpointIpv4,
			y:    secondEndpointIpv4,
			want: true,
		},
		{
			name: "identical ipv6",
			x:    firstEndpointIpv6,
			y:    secondEndpointIpv6,
			want: true,
		},
		{
			name: "different ipv4",
			x:    firstEndpointIpv4,
			y:    thirdEndpointIpv4,
			want: false,
		},
		{
			name: "different ipv6",
			x:    firstEndpointIpv6,
			y:    thirdEndpointIpv6,
			want: false,
		},
		{
			name: "different ipv4 and ipv6",
			x:    firstEndpointIpv4,
			y:    firstEndpointIpv6,
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

func TestGetIPTypeFromString(t *testing.T) {
	tests := []struct {
		name      string
		ipTypeStr string
		want      IPType
		wantErr   bool
	}{
		{
			name:      "happy case ipv4",
			ipTypeStr: "IPV4Type",
			want:      IPV4Type,
			wantErr:   false,
		},
		{
			name:      "happy case ipv6",
			ipTypeStr: "IPV6Type",
			want:      IPV6Type,
			wantErr:   false,
		},
		{
			name:      "empty string",
			ipTypeStr: "",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "case wrong",
			ipTypeStr: "ipv6type",
			want:      "",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetIPTypeFromString(tt.ipTypeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIPTypeFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetIPTypeFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
