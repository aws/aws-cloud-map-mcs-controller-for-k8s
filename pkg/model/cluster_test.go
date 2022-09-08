package model

import (
	"context"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	aboutv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/about/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestClusterUtils_GetClusterProperties(t *testing.T) {
	type fields struct {
		client            client.Client
		clusterProperties clusterProperties
	}
	type args struct {
		ctx context.Context
	}
	clusterId := "cluster1"
	clusterSetId := "clusterset1"
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *clusterProperties
		wantErr bool
	}{
		{
			name: "happy case fetch from client",
			fields: fields{
				client:            fake.NewClientBuilder().WithScheme(GetScheme()).WithObjects(ClusterIdForTest(clusterId), ClusterSetIdForTest(clusterSetId)).Build(),
				clusterProperties: clusterProperties{},
			},
			args:    args{ctx: context.TODO()},
			want:    &clusterProperties{clusterId: clusterId, clusterSetId: clusterSetId},
			wantErr: false,
		},
		{
			name: "happy case already set",
			fields: fields{
				client:            nil,
				clusterProperties: clusterProperties{clusterId: clusterId, clusterSetId: clusterSetId},
			},
			args:    args{ctx: context.TODO()},
			want:    &clusterProperties{clusterId: clusterId, clusterSetId: clusterSetId},
			wantErr: false,
		},
		{
			name: "error cluster properties not present",
			fields: fields{
				client:            fake.NewClientBuilder().WithScheme(GetScheme()).Build(),
				clusterProperties: clusterProperties{},
			},
			args:    args{ctx: context.TODO()},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ClusterUtils{
				client:            tt.fields.client,
				clusterProperties: tt.fields.clusterProperties,
			}
			got, err := r.GetClusterProperties(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClusterProperties() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClusterProperties() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func ClusterIdForTest(clusterId string) *aboutv1alpha1.ClusterProperty {
	return &aboutv1alpha1.ClusterProperty{
		ObjectMeta: metav1.ObjectMeta{
			Name: ClusterIdPropertyName,
		},
		Spec: aboutv1alpha1.ClusterPropertySpec{
			Value: clusterId,
		},
	}
}

func ClusterSetIdForTest(clusterSetId string) *aboutv1alpha1.ClusterProperty {
	return &aboutv1alpha1.ClusterProperty{
		ObjectMeta: metav1.ObjectMeta{
			Name: ClusterSetIdPropertyName,
		},
		Spec: aboutv1alpha1.ClusterPropertySpec{
			Value: clusterSetId,
		},
	}
}

func GetScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(aboutv1alpha1.GroupVersion, &aboutv1alpha1.ClusterProperty{}, &aboutv1alpha1.ClusterPropertyList{})
	return scheme
}
