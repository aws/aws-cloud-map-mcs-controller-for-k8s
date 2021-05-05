package model

type Plan struct {
	// List of current instances
	Current []*Endpoint

	// List of desired instances
	Desired []*Endpoint
}

type Changes struct {
	// List of endpoints that need to be created
	Create []*Endpoint
	// List of endpoints that need to be updated
	Update []*Endpoint
	// List of endpoints that need to be deleted
	Delete []*Endpoint
}

// CalculateChanges returns list of Changes that need to applied
func (p *Plan) CalculateChanges() Changes {
	changes := Changes{}

	currentMap := make(map[string]*Endpoint)
	for _, e := range p.Current {
		currentMap[e.Id] = e
	}

	for _, e := range p.Desired {
		existing := currentMap[e.Id]
		if existing != nil {
			if !existing.Equals(e) {
				changes.Update = append(changes.Update, e)
			}
			delete(currentMap, e.Id)
		} else {
			changes.Create = append(changes.Create, e)
		}
	}

	// iterate unmatched endpoints from Current to delete them
	for _, e := range currentMap {
		changes.Delete = append(changes.Delete, e)
	}

	return changes
}
