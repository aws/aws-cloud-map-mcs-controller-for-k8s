package model

import (
	"reflect"
	"testing"
)

func TestPlan_CalculateChanges(t *testing.T) {
	type fields struct {
		Current []*Endpoint
		Desired []*Endpoint
	}
	tests := []struct {
		name   string
		fields fields
		want   Changes
	}{
		{
			name: "No changes",
			fields: fields{
				Current: []*Endpoint{{Id: "inst-1"}},
				Desired: []*Endpoint{{Id: "inst-1"}},
			},
			want: Changes{},
		},
		{
			name: "New endpoint added",
			fields: fields{
				Current: []*Endpoint{{Id: "inst-1"}},
				Desired: []*Endpoint{{Id: "inst-1"}, {Id: "inst-2"}},
			},
			want: Changes{
				Create: []*Endpoint{{Id: "inst-2"}},
			},
		},
		{
			name: "Endpoint deleted",
			fields: fields{
				Current: []*Endpoint{{Id: "inst-1"}, {Id: "inst-2"}},
				Desired: []*Endpoint{{Id: "inst-1"}},
			},
			want: Changes{
				Delete: []*Endpoint{{Id: "inst-2"}},
			},
		},
		{
			name: "Endpoint updated",
			fields: fields{
				Current: []*Endpoint{{Id: "inst-1", IP: "1.1.1.1"}},
				Desired: []*Endpoint{{Id: "inst-1", IP: "1.1.1.2"}},
			},
			want: Changes{
				Update: []*Endpoint{{Id: "inst-1", IP: "1.1.1.2"}},
			},
		},
		{
			name: "Endpoint added/deleted/updated at the same time",
			fields: fields{
				Current: []*Endpoint{{Id: "inst-1", IP: "1.1.1.1"}, {Id: "inst-2", IP: "1.1.1.2"}},
				Desired: []*Endpoint{{Id: "inst-3", IP: "1.1.1.3"}, {Id: "inst-2", IP: "2.2.2.2"}},
			},
			want: Changes{
				Delete: []*Endpoint{{Id: "inst-1", IP: "1.1.1.1"}},
				Create: []*Endpoint{{Id: "inst-3", IP: "1.1.1.3"}},
				Update: []*Endpoint{{Id: "inst-2", IP: "2.2.2.2"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plan{
				Current: tt.fields.Current,
				Desired: tt.fields.Desired,
			}
			if got := p.CalculateChanges(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CalculateChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}
