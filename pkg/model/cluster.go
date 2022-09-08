package model

import (
	"context"
	"fmt"

	aboutv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/about/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterIdPropertyName    = "cluster.clusterset.k8s.io"
	ClusterSetIdPropertyName = "clusterset.k8s.io"
)

// Non-exported type, accessible via read-only func
type clusterProperties struct {
	clusterId    string
	clusterSetId string
}

func (r clusterProperties) ClusterId() string {
	return r.clusterId
}

func (r clusterProperties) ClusterSetId() string {
	return r.clusterSetId
}

func (r clusterProperties) IsValid() bool {
	return r.clusterSetId != "" && r.clusterId != ""
}

func (r clusterProperties) String() string {
	return fmt.Sprintf("ClusterId: %s, ClusterSetId: %s", r.clusterId, r.clusterSetId)
}

// ClusterUtils provides utility functions for working with clusters
type ClusterUtils struct {
	client            client.Client
	clusterProperties clusterProperties
}

func NewClusterUtils(client client.Client) ClusterUtils {
	return ClusterUtils{
		client:            client,
		clusterProperties: clusterProperties{},
	}
}

func NewClusterUtilsWithValues(clusterId string, clusterSetId string) ClusterUtils {
	return ClusterUtils{
		clusterProperties: clusterProperties{clusterId: clusterId, clusterSetId: clusterSetId},
	}
}

func (r *ClusterUtils) GetClusterProperties(ctx context.Context) (*clusterProperties, error) {
	if !r.clusterProperties.IsValid() {
		err := r.LoadClusterProperties(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &r.clusterProperties, nil
}

func (r *ClusterUtils) LoadClusterProperties(ctx context.Context) error {
	clusterPropertyList := &aboutv1alpha1.ClusterPropertyList{}
	err := r.client.List(ctx, clusterPropertyList)
	if err != nil {
		return err
	}
	for _, clusterProperty := range clusterPropertyList.Items {
		switch clusterProperty.Name {
		case ClusterIdPropertyName:
			r.clusterProperties.clusterId = clusterProperty.Spec.Value
		case ClusterSetIdPropertyName:
			r.clusterProperties.clusterSetId = clusterProperty.Spec.Value
		}
	}
	if !r.clusterProperties.IsValid() {
		return fmt.Errorf("ClusterProperty not found: %s", r.clusterProperties)
	}
	return nil
}
