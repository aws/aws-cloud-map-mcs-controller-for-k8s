package model

import (
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"reflect"
	"testing"
)

var instId = "my-instance"
var ip = "192.168.0.1"

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
					Ipv4Attr:      ip,
					PortAttr:      "65535",
					"custom-attr": "custom-val",
				},
			},
			want: &Endpoint{
				Id:   instId,
				IP:   ip,
				Port: 65535,
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
					Ipv4Attr:      ip,
					PortAttr:      "99999",
					"custom-attr": "custom-val",
				},
			},
			wantErr: true,
		},
		{
			name: "missing IP",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					PortAttr:      "80",
					"custom-attr": "custom-val",
				},
			},
			wantErr: true,
		},
		{
			name: "missing port",
			inst: &types.HttpInstanceSummary{
				InstanceId: &instId,
				Attributes: map[string]string{
					Ipv4Attr:      ip,
					"custom-attr": "custom-val",
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
	type fields struct {
		Id         string
		IP         string
		Port       int32
		Attributes map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "happy case",
			fields: fields{
				IP:   ip,
				Port: 30,
				Attributes: map[string]string{
					"custom-attr": "custom-val",
				},
			},
			want: map[string]string{
				Ipv4Attr:      ip,
				PortAttr:      "30",
				"custom-attr": "custom-val",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Endpoint{
				Id:         tt.fields.Id,
				IP:         tt.fields.IP,
				Port:       tt.fields.Port,
				Attributes: tt.fields.Attributes,
			}
			if got := e.GetCloudMapAttributes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpointIdFromIPAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{
			name:    "happy case",
			address: "192.168.0.1",
			want:    "192_168_0_1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EndpointIdFromIPAddress(tt.address); got != tt.want {
				t.Errorf("EndpointIdFromIPAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpoint_Equals(t *testing.T) {
	firstEndpoint := Endpoint{
		Id:   instId,
		IP:   ip,
		Port: 80,
		Attributes: map[string]string{
			"custom-key": "custom-val",
		},
	}

	secondEndpoint := Endpoint{
		Id:   instId,
		IP:   ip,
		Port: 80,
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
			y:    firstEndpoint,
			want: true,
		},
		{
			name: "different",
			x:    firstEndpoint,
			y:    secondEndpoint,
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
