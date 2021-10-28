package main

import (
	"context"
)

const (
	e2eNs = "aws-cloud-map-mcs-e2e"
)

func main() {
	j := NewDefaultJanitor()
	j.Cleanup(context.TODO(), e2eNs)
}
