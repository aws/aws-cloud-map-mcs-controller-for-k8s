package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/integration/shared/scenarios"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	if len(os.Args) != 9 {
		fmt.Println("Expected namespace, service, clusterId, clusterSetId, endpoint port, service port, endpoint IP list, and serviceType arguments")
		os.Exit(1)
	}

	nsName := os.Args[1]
	svcName := os.Args[2]
	clusterId := os.Args[3]
	clusterSetId := os.Args[4]
	port := os.Args[5]
	servicePort := os.Args[6]
	ips := os.Args[7]
	serviceType := os.Args[8]

	testServiceExport(nsName, svcName, clusterId, clusterSetId, port, servicePort, ips, serviceType)
}

func testServiceExport(nsName string, svcName string, clusterId string, clusterSetId string, port string, servicePort string, ips string, serviceType string) {
	fmt.Printf("Testing service export integration for namespace %s and service %s\n", nsName, svcName)

	export, err := scenarios.NewExportServiceScenario(getAwsConfig(), nsName, svcName, clusterId, clusterSetId, port, servicePort, ips, serviceType)
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
	awsCfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		fmt.Printf("unable to configure AWS session: %s", err.Error())
		os.Exit(1)
	}

	return &awsCfg
}
