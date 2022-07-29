package common

import (
	"context"
	"fmt"

	aboutv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/about/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterIdName    = "id.k8s.io"
	ClusterSetIdName = "clusterset.k8s.io"
)

// ClusterUtils provides utility functions for working with clusters
type ClusterUtils struct {
	client       client.Client
	clusterId    string
	clusterSetId string
}

// constructor
func NewClusterUtils(client client.Client) ClusterUtils {
	return ClusterUtils{
		client: client,
	}
}

// retrieve the clusterId from the local field. If not set, retrieve from client
func (r *ClusterUtils) GetClusterId(ctx context.Context) (string, error) {
	if r.clusterId != "" {
		return r.clusterId, nil
	}
	clusterPropertyForClusterId := &aboutv1alpha1.ClusterProperty{}
	err := r.client.Get(ctx, client.ObjectKey{Name: ClusterIdName}, clusterPropertyForClusterId)
	if err != nil {
		return "", err
	}
	if clusterPropertyForClusterId.Spec.Value == "" {
		err := fmt.Errorf("ClusterId not found")
		return "", err
	}
	r.clusterId = clusterPropertyForClusterId.Spec.Value
	return r.clusterId, nil
}

// retrieve the clusterSetId from the local field. If not set, retrieve from client
func (r *ClusterUtils) GetClusterSetId(ctx context.Context) (string, error) {
	if r.clusterSetId != "" {
		return r.clusterSetId, nil
	}
	clusterPropertyForClusterSetId := &aboutv1alpha1.ClusterProperty{}
	err := r.client.Get(ctx, client.ObjectKey{Name: ClusterSetIdName}, clusterPropertyForClusterSetId)
	if err != nil {
		return "", err
	}
	if clusterPropertyForClusterSetId.Spec.Value == "" {
		err := fmt.Errorf("ClusterSetId not found")
		return "", err
	}
	r.clusterSetId = clusterPropertyForClusterSetId.Spec.Value
	return r.clusterSetId, nil
}
