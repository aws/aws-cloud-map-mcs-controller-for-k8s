package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/integration/janitor"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Expected single namespace name argument")
		os.Exit(1)
	}

	j := janitor.NewDefaultJanitor()
	nsName := os.Args[1]
	j.Cleanup(context.TODO(), nsName)
}
