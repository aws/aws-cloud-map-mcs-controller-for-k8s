package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/integration/scenarios"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"os"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Println("Expected namespace, service, endpoint port, and endpoint IP list arguments")
		os.Exit(1)
	}

	nsName := os.Args[1]
	svcName := os.Args[2]
	port := os.Args[3]
	ips := os.Args[4]

	testServiceExport(nsName, svcName, port, ips)
}

func testServiceExport(nsName string, svcName string, port string, ips string) {
	fmt.Printf("Testing service export integration for namespace %s and service %s\n", nsName, svcName)

	export, err := scenarios.NewExportServiceScenario(getAwsConfig(), nsName, svcName, port, ips)
	if err != nil {
		fmt.Printf("Failed to setup service export integration test scenario: %s", err.Error())
		os.Exit(1)
	}

	if err := export.Run(); err != nil {
		fmt.Printf("Service export integration test scenario failed: %s", err.Error())
		os.Exit(1)
	}
}

func getAwsConfig() *aws.Config {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))

	if err != nil {
		fmt.Printf("unable to configure AWS session: %s", err.Error())
		os.Exit(1)
	}

	return &awsCfg
}
