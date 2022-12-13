package model

import (
	"reflect"
	"strconv"
	"testing"

	discovery "k8s.io/api/discovery/v1"

	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/google/go-cmp/cmp"
)

var instId = "my-instance"
var ipv4 = "192.168.0.1"
var ipv6 = "2001:0db8:0001:0000:0000:0ab9:C0A8:0102"
var clusterId = "test-mcs-clusterId"
var clusterId2 = "test-mcs-clusterid-2"
var clusterId3 = "test-mcs-clusterid-3"
var namespaceName = "test-mcs-namespace"
var serviceName = "test-mcs-service"
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
				Id:          instId,
				IP:          ipv4,
				AddressType: discovery.AddressTypeIPv4,
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
				Id:          instId,
				IP:          ipv6,
				AddressType: discovery.AddressTypeIPv6,
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
			name: "ipv4 and ipv6 defaults to ipv4",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					ClusterIdAttr:             clusterId,
					ClusterSetIdAttr:          clusterSetId,
					EndpointIpv4Attr:          ipv4,
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
				Id:          instId,
				IP:          ipv4,
				AddressType: discovery.AddressTypeIPv4,
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
				IP:          ipv4,
				AddressType: discovery.AddressTypeIPv4,
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
				IP:          ipv6,
				AddressType: discovery.AddressTypeIPv6,
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
		Id:          instId,
		IP:          ipv4,
		AddressType: discovery.AddressTypeIPv4,
		ServicePort: Port{
			Port: 80,
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	secondEndpointIpv4 := Endpoint{
		Id:          instId,
		IP:          ipv4,
		AddressType: discovery.AddressTypeIPv4,
		ServicePort: Port{
			Port: 80,
			Name: "",
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	thirdEndpointIpv4 := Endpoint{
		Id:          instId,
		IP:          ipv4,
		AddressType: discovery.AddressTypeIPv4,
		ServicePort: Port{
			Port: 80,
		},
		Attributes: map[string]string{
			"custom-key": "different-val",
		},
	}

	firstEndpointIpv6 := Endpoint{
		Id:          instId,
		IP:          ipv6,
		AddressType: discovery.AddressTypeIPv6,
		ServicePort: Port{
			Port: 80,
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	secondEndpointIpv6 := Endpoint{
		Id:          instId,
		IP:          ipv6,
		AddressType: discovery.AddressTypeIPv6,
		ServicePort: Port{
			Port: 80,
			Name: "",
		},
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	thirdEndpointIpv6 := Endpoint{
		Id:          instId,
		IP:          ipv6,
		AddressType: discovery.AddressTypeIPv6,
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

func TestGetAddressTypeFromString(t *testing.T) {
	tests := []struct {
		name           string
		addressTypeStr string
		want           discovery.AddressType
		wantErr        bool
	}{
		{
			name:           "happy case ipv4",
			addressTypeStr: "IPv4",
			want:           discovery.AddressTypeIPv4,
			wantErr:        false,
		},
		{
			name:           "happy case ipv6",
			addressTypeStr: "IPv6",
			want:           discovery.AddressTypeIPv6,
			wantErr:        false,
		},
		{
			name:           "empty string",
			addressTypeStr: "",
			want:           "",
			wantErr:        true,
		},
		{
			name:           "case wrong",
			addressTypeStr: "IPV6",
			want:           "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAddressTypeFromString(tt.addressTypeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAddressTypeFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetAddressTypeFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEndpoints(t *testing.T) {
	firstEndpoint := Endpoint{
		Id:          instId + "-1",
		IP:          ipv4,
		AddressType: discovery.AddressTypeIPv4,
		ServicePort: Port{
			Port: 80,
		},
		ClusterId: clusterId,
	}
	secondEndpoint := Endpoint{
		Id:          instId + "2",
		IP:          ipv4,
		AddressType: discovery.AddressTypeIPv4,
		ServicePort: Port{
			Port: 80,
			Name: "",
		},
		ClusterId: clusterId2,
	}
	thirdEndpoint := Endpoint{
		Id:          instId + "3",
		IP:          ipv4,
		AddressType: discovery.AddressTypeIPv4,
		ServicePort: Port{
			Port: 80,
			Name: "",
		},
		ClusterId: clusterId2,
	}

	svc := Service{
		Namespace: namespaceName,
		Name:      serviceName,
		Endpoints: []*Endpoint{
			&firstEndpoint, &secondEndpoint, &thirdEndpoint,
		},
	}

	tests := []struct {
		name       string
		x          string
		wantEndpts []*Endpoint
	}{
		{
			name:       "return-first-endpoint",
			x:          clusterId,
			wantEndpts: []*Endpoint{&firstEndpoint},
		},
		{
			name:       "return-two-endpoints",
			x:          clusterId2,
			wantEndpts: []*Endpoint{&secondEndpoint, &thirdEndpoint},
		},
		{
			name:       "return-nil",
			x:          clusterId3,
			wantEndpts: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEndpts := svc.GetEndpoints(tt.x); !cmp.Equal(gotEndpts, tt.wantEndpts) {
				t.Errorf("Equals() = %v, Want = %v", gotEndpts, tt.wantEndpts)
			}
		})
	}
}
