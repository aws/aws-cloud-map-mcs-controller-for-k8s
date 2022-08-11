package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/integration/janitor"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Expected namespace name, clusterId, clusterSetId arguments")
		os.Exit(1)
	}

	nsName := os.Args[1]
	clusterId := os.Args[2]
	clusterSetId := os.Args[3]

	j := janitor.NewDefaultJanitor(clusterId, clusterSetId)
	j.Cleanup(context.TODO(), nsName)
}
