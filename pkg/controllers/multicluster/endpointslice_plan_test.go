package controllers

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/stretchr/testify/assert"
	discovery "k8s.io/api/discovery/v1beta1"
)

func TestEndpointSlicePlan_CalculateChanges(t *testing.T) {
	type fields struct {
		Current []*discovery.EndpointSlice
		Desired []*model.Endpoint
	}
	tests := []struct {
		name   string
		fields fields
		want   EndpointSliceChanges
	}{
		{
			name: "no changes",
			fields: fields{
				Current: []*discovery.EndpointSlice{endpointSliceForTest()},
				Desired: []*model.Endpoint{test.GetTestEndpoint1()},
			},
			want: EndpointSliceChanges{
				Unmodified: []*discovery.EndpointSlice{endpointSliceForTest()},
			},
		},
		{
			name: "delete slice",
			fields: fields{
				Current: []*discovery.EndpointSlice{endpointSliceForTest()},
				Desired: []*model.Endpoint{},
			},
			want: EndpointSliceChanges{
				Delete: []*discovery.EndpointSlice{endpointSliceForTest()},
			},
		},
		{
			name: "new slice",
			fields: fields{
				Current: []*discovery.EndpointSlice{},
				Desired: []*model.Endpoint{test.GetTestEndpoint1()},
			},
			want: EndpointSliceChanges{
				Create: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp1},
						[]discovery.EndpointPort{
							PortToEndpointPort(test.GetTestEndpoint1().EndpointPort),
						},
					),
				},
			},
		},
		{
			name: "removed endpoint needs slice update",
			fields: fields{
				Current: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp1, test.EndptIp2},
						[]discovery.EndpointPort{
							PortToEndpointPort(test.GetTestEndpoint1().EndpointPort),
						},
					),
				},
				Desired: []*model.Endpoint{test.GetTestEndpoint2()},
			},
			want: EndpointSliceChanges{
				Update: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp2},
						[]discovery.EndpointPort{
							PortToEndpointPort(test.GetTestEndpoint2().EndpointPort),
						},
					),
				},
			},
		},
		{
			name: "added endpoint needs slice update",
			fields: fields{
				Current: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp1},
						[]discovery.EndpointPort{
							PortToEndpointPort(model.Port{Name: test.PortName1, Port: test.Port1, Protocol: test.Protocol1}),
						},
					),
				},
				Desired: []*model.Endpoint{
					test.GetTestEndpoint1(),
					{
						Id: test.EndptId2,
						IP: test.EndptIp2,
						EndpointPort: model.Port{
							Name:     test.PortName1,
							Port:     test.Port1,
							Protocol: test.Protocol1,
						},
					},
				},
			},
			want: EndpointSliceChanges{
				Update: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp1, test.EndptIp2},
						[]discovery.EndpointPort{
							PortToEndpointPort(test.GetTestEndpoint1().EndpointPort),
						},
					),
				},
				Unmodified: []*discovery.EndpointSlice{},
			},
		},
		{
			name: "swapped endpoints need slice update",
			fields: fields{
				Current: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp1},
						[]discovery.EndpointPort{
							PortToEndpointPort(test.GetTestEndpoint2().EndpointPort),
						},
					),
				},
				Desired: []*model.Endpoint{
					test.GetTestEndpoint2(),
				},
			},
			want: EndpointSliceChanges{
				Update: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp2},
						[]discovery.EndpointPort{
							PortToEndpointPort(test.GetTestEndpoint2().EndpointPort),
						},
					),
				},
				Delete: []*discovery.EndpointSlice{},
			},
		},
		{
			name: "changed ports need slice update",
			fields: fields{
				Current: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp2},
						[]discovery.EndpointPort{
							PortToEndpointPort(test.GetTestEndpoint1().EndpointPort),
						},
					),
				},
				Desired: []*model.Endpoint{
					test.GetTestEndpoint2(),
				},
			},
			want: EndpointSliceChanges{
				Update: []*discovery.EndpointSlice{
					endpointSliceWithIpsAndPortsForTest(
						[]string{test.EndptIp2},
						[]discovery.EndpointPort{
							PortToEndpointPort(test.GetTestEndpoint2().EndpointPort),
						},
					),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &EndpointSlicePlan{
				Service:           k8sServiceForTest(),
				ServiceImportName: test.SvcName,
				ClusterId:         test.ClusterId1,
				Current:           tt.fields.Current,
				Desired:           tt.fields.Desired,
			}
			if got := p.CalculateChanges(); !reflect.DeepEqual(got, tt.want) {
				gotJson, _ := json.MarshalIndent(got, "", "  ")
				wantJson, _ := json.MarshalIndent(tt.want, "", "  ")
				t.Errorf("CalculateChanges() = \n%s\nwant = \n%s", gotJson, wantJson)
			}
		})
	}
}

func TestEndpointSlicePlan_MultipleSliceCreation(t *testing.T) {
	p := &EndpointSlicePlan{
		maxEndpointsPerSlice: 2,
		Service:              k8sServiceForTest(),
		ServiceImportName:    test.SvcName,
		ClusterId:            test.ClusterId1,
		Current:              []*discovery.EndpointSlice{},
		Desired:              test.GetTestEndpoints(43),
	}
	changes := p.CalculateChanges()
	assert.Equal(t, 22, len(changes.Create))
}

func TestEndpointSlicePlan_PreferCreateOverMultipleSliceUpdate(t *testing.T) {
	p := &EndpointSlicePlan{
		maxEndpointsPerSlice: 2,
		Service:              k8sServiceForTest(),
		ServiceImportName:    test.SvcName,
		ClusterId:            test.ClusterId1,
		Current:              []*discovery.EndpointSlice{endpointSliceForTest()},
		Desired:              []*model.Endpoint{test.GetTestEndpoint1()},
	}
	p.Desired = append(p.Desired, test.GetTestEndpoints(2)...)
	changes := p.CalculateChanges()
	assert.Equal(t, 1, len(changes.Create))
	assert.Equal(t, 1, len(changes.Unmodified))
	assert.Equal(t, 0, len(changes.Update))
	assert.Equal(t, 0, len(changes.Delete))
}
