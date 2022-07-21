package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/integration/janitor"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Expected single namespace name and clustersetId arguments")
		os.Exit(1)
	}

	j := janitor.NewDefaultJanitor()
	nsName := os.Args[1]
	clustersetId := os.Args[2]
	j.Cleanup(context.TODO(), nsName, clustersetId)
}
