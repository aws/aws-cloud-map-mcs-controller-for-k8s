package main

import (
	"context"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/integration/janitor"
)

const (
	e2eNs = "aws-cloud-map-mcs-e2e"
)

func main() {
	j := janitor.NewDefaultJanitor()
	j.Cleanup(context.TODO(), e2eNs)
}
