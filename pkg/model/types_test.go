package model

import (
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"reflect"
	"testing"
)

var instId = "my-instance"

func TestNewEndpointFromInstance(t *testing.T) {
	tests := []struct {
		name    string
		inst    *types.InstanceSummary
		want    *Endpoint
		wantErr bool
	}{
		{
			name: "happy case",
			inst: &types.InstanceSummary{
				Id: &instId,
				Attributes: map[string]string{
					Ipv4Attr:      "192.168.0.1",
					PortAttr:      "65535",
					"custom-attr": "custom-val",
				},
			},
			want: &Endpoint{
				Id:   instId,
				IP:   "192.168.0.1",
				Port: 65535,
				Attributes: map[string]string{
					"custom-attr": "custom-val",
				},
			},
		},
		{
			name: "invalid port",
			inst: &types.InstanceSummary{
				Id: &instId,
				Attributes: map[string]string{
					Ipv4Attr:      "192.168.0.1",
					PortAttr:      "99999",
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
