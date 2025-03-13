package controllers

import (
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
)

const defaultMaxEndpointsPerSlice = 100

type EndpointSliceChanges struct {
	// Create: List of EndpointSlices that need to be created
	Create []*discovery.EndpointSlice
	// Update: List of EndpointSlices that need to be updated
	Update []*discovery.EndpointSlice
	// Delete: List of EndpointSlices that need to be deleted
	Delete []*discovery.EndpointSlice
	// Unmodified: List of EndpointSlices that do not need to be changed
	Unmodified []*discovery.EndpointSlice
}

type EndpointSlicePlan struct {
	// maxEndpointsPerSlice defaults to 100
	maxEndpointsPerSlice int

	// Service to reconcile endpoints in
	Service *v1.Service

	// ServiceImportName name used to create new EndpointSlices
	ServiceImportName string

	// Current EndpontSlices
	Current []*discovery.EndpointSlice

	// Desired Endpoints
	Desired []*model.Endpoint

	// Cluster the EndpointSlice belongs to
	ClusterId string
}

// CheckAddressType TODO: Will need to improve how IP Type is determined when we implement dual stack.
func (p *EndpointSlicePlan) CheckAddressType() discovery.AddressType {
	// Peek at the first endpoint for its AddressType. All endpoints in a slice will be of the same AddressType.
	if len(p.Desired) == 0 {
		return discovery.AddressTypeIPv4
	}
	return p.Desired[0].AddressType
}

// CalculateChanges returns list of EndpointSlice Changes that need to applied
func (p *EndpointSlicePlan) CalculateChanges() EndpointSliceChanges {
	// populate map of desired endpoints for lookup efficiency
	desiredEndpoints := make(map[string]*model.Endpoint)
	for _, desiredEndpoint := range p.Desired {
		desiredEndpoints[desiredEndpoint.IP] = desiredEndpoint
	}

	desiredPorts := ExtractEndpointPorts(p.Desired)

	// Remove unwanted endpoints from slices
	changes := p.trimSlices(desiredEndpoints, desiredPorts)

	// Add new endpoints to slices
	for len(desiredEndpoints) > 0 {
		sliceWithRoom, needsPortUpdate := p.getOrCreateUnfilledEndpointSlice(&changes, len(desiredEndpoints))

		for key, endpointToAdd := range desiredEndpoints {
			roomInSlice := p.getMaxEndpointsPerSlice() - len(sliceWithRoom.Endpoints)
			if roomInSlice <= 0 {
				// stop adding to slice once it is full
				break
			}
			sliceWithRoom.Endpoints = append(sliceWithRoom.Endpoints, CreateEndpointForSlice(p.Service, endpointToAdd))
			delete(desiredEndpoints, key)
		}

		if needsPortUpdate {
			newPorts := portSliceToEndpointPortSlice(desiredPorts)
			sliceWithRoom.Ports = newPorts
		}
	}

	return changes
}

func (p *EndpointSlicePlan) trimSlices(desiredEndpoints map[string]*model.Endpoint, desiredPorts []*model.Port) (changes EndpointSliceChanges) {
	// remove all undesired existing endpoints in slices
	for _, existingSlice := range p.Current {
		updatedEndpointList := make([]discovery.Endpoint, 0)

		sliceNeedsUpdateConditions := false

		for _, existingEndpoint := range existingSlice.Endpoints {
			key := existingEndpoint.Addresses[0]

			if _, found := desiredEndpoints[key]; found {
				// if different ready status, set sliceNeedsUpdateConditions to true
				if existingEndpoint.Conditions.Ready != &desiredEndpoints[key].Ready {
					sliceNeedsUpdateConditions = true
					existingEndpoint.Conditions.Ready = &desiredEndpoints[key].Ready
				}
				updatedEndpointList = append(updatedEndpointList, existingEndpoint)
				delete(desiredEndpoints, key)
			}
		}

		// mark slice for deletion if all endpoints were removed
		if len(updatedEndpointList) == 0 {
			changes.Delete = append(changes.Delete, existingSlice)
			continue
		}

		sliceNeedsUpdate := false

		// slice needs to be updated if ports do not match
		if !PortsEqualIgnoreOrder(desiredPorts, endpointPortSliceToPortSlice(existingSlice.Ports)) {
			existingSlice.Ports = portSliceToEndpointPortSlice(desiredPorts)
			sliceNeedsUpdate = true
		}

		// slice needs to be updated if endpoint list changed
		if len(updatedEndpointList) != len(existingSlice.Endpoints) || sliceNeedsUpdateConditions {
			existingSlice.Endpoints = updatedEndpointList
			sliceNeedsUpdate = true
		}

		if sliceNeedsUpdate {
			changes.Update = append(changes.Update, existingSlice)
		} else {
			changes.Unmodified = append(changes.Unmodified, existingSlice)
		}
	}

	return changes
}

func (p *EndpointSlicePlan) getOrCreateUnfilledEndpointSlice(changes *EndpointSliceChanges, requiredCapacity int) (sliceWithRoom *discovery.EndpointSlice, needsPortUpdate bool) {
	// Prefer slices we are already updating
	for _, sliceToUpdate := range changes.Update {
		if len(sliceToUpdate.Endpoints) < p.getMaxEndpointsPerSlice() {
			return sliceToUpdate, false
		}
	}

	// Update a slice marked for deletion if possible
	if len(changes.Delete) > 0 {
		sliceToReuse := changes.Delete[0]
		changes.Delete = changes.Delete[1:]
		changes.Update = append(changes.Update, sliceToReuse)

		// clear endpoint list that was marked for deletion before reusing
		sliceToReuse.Endpoints = []discovery.Endpoint{}
		return sliceToReuse, true
	}

	// Update an unmodified slice if it has capacity to add all endpoints
	for i, unmodifiedSlice := range changes.Unmodified {
		proposedSliceLength := len(unmodifiedSlice.Endpoints) + requiredCapacity
		if proposedSliceLength <= p.getMaxEndpointsPerSlice() {
			changes.Unmodified = append(changes.Unmodified[:i], changes.Unmodified[i+1:]...)
			changes.Update = append(changes.Update, unmodifiedSlice)
			return unmodifiedSlice, false
		}
	}

	// No existing slices can fill new endpoint requirements so create a new slice
	sliceToCreate := CreateEndpointSliceStruct(p.Service, p.ServiceImportName, p.ClusterId, p.CheckAddressType())
	changes.Create = append(changes.Create, sliceToCreate)
	return sliceToCreate, true
}

func (p *EndpointSlicePlan) getMaxEndpointsPerSlice() int {
	if p.maxEndpointsPerSlice != 0 {
		return p.maxEndpointsPerSlice
	}

	return defaultMaxEndpointsPerSlice
}

func endpointPortSliceToPortSlice(endpointPorts []discovery.EndpointPort) (ports []*model.Port) {
	for _, endpointPort := range endpointPorts {
		port := EndpointPortToPort(endpointPort)
		ports = append(ports, &port)
	}
	return ports
}

func portSliceToEndpointPortSlice(ports []*model.Port) (endpointPorts []discovery.EndpointPort) {
	for _, port := range ports {
		endpointPort := PortToEndpointPort(*port)
		endpointPorts = append(endpointPorts, endpointPort)
	}
	return endpointPorts
}
