package cloudmap

import (
	"github.com/aws/aws-k8s-mcs-controller/pkg/model"
	"github.com/go-logr/logr"
)

type Client interface {
	// CreateService creates a Cloud Map service resource and return created service struct
	CreateService(service *model.Service) error

	// GetService returns a service resource fetched from the Cloud Map API or nil if not found
	GetService(namespace string, name string) (*model.Service, error)

	// RegisterEndpoints registers all endpoints for given service
	RegisterEndpoints(service *model.Service) error

	// DeleteEndpoints de-registers all endpoints for given service
	DeleteEndpoints(service *model.Service) error
}

// TODO replace by Cloud Map API implementation
type ClientMock struct {
	Log logr.Logger
}

func (c *ClientMock) CreateService(service *model.Service) error {
	c.Log.Info("creating a new service", "namespace", service.Namespace, "name", service.Name)
	return nil
}

func (c *ClientMock) GetService(namespace string, name string) (*model.Service, error) {
	c.Log.Info("fetching a service", "namespace", namespace, "name", name)
	return nil, nil
}

func (c *ClientMock) RegisterEndpoints(service *model.Service) error {
	for _, e := range service.Endpoints {
		c.Log.Info("registering endpoint",
			"namespace", service.Namespace, "name", service.Name, "endpoint", e)
	}
	return nil
}

func (c *ClientMock) DeleteEndpoints(service *model.Service) error {
	c.Log.Info("deleting all endpoints for a given service",
		"namespace", service.Namespace, "name", service.Name)
	return nil
}
