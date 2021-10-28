package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	sd "github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"os"
)

const (
	e2eNs = "aws-cloud-map-mcs-e2e"
)

type Janitor interface {
	Cleanup(ctx context.Context, nsName string)
}

type janitor struct {
	awsSdk cloudmap.AwsFacade
	sdApi  cloudmap.ServiceDiscoveryApi
	fail   func()
}

func main() {
	j := newDefaultJanitor()
	j.Cleanup(context.TODO(), e2eNs)
}

func newDefaultJanitor() Janitor {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)

	if err != nil {
		fmt.Printf("unable to configure AWS session: %s", err.Error())
		os.Exit(1)
	}

	return &janitor{
		awsSdk: cloudmap.NewAwsFacadeFromConfig(&awsCfg),
		sdApi:  cloudmap.NewServiceDiscoveryApiFromConfig(&awsCfg),
		fail: func() {
			os.Exit(1)
		},
	}
}

func (j *janitor) Cleanup(ctx context.Context, nsName string) {
	fmt.Printf("Cleaning up all test resources in Cloud Map for namespace : %s\n", nsName)

	nsId, err := j.getNs(ctx, nsName)
	j.checkOrFail(err, fmt.Sprintf("found namespace to clean: %s", nsId), "could not find namespace to clean")

	svcs, err := j.sdApi.ListServices(ctx, nsId)
	j.checkOrFail(err,
		fmt.Sprintf("namespace has %d services to clean", len(svcs)),
		"could not find services to clean")

	for _, svc := range svcs {
		fmt.Printf("found service to clean: %s\n", svc.Id)
		j.deregInsts(ctx, svc.Id)

		_, delSvcErr := j.awsSdk.DeleteService(ctx, &sd.DeleteServiceInput{Id: &svc.Id})
		j.checkOrFail(delSvcErr, "service deleted", "could not cleanup service")
	}

	out, err := j.awsSdk.DeleteNamespace(ctx, &sd.DeleteNamespaceInput{Id: &nsId})
	if err == nil {
		_, err = j.sdApi.PollNamespaceOperation(ctx, aws.ToString(out.OperationId))
	}
	j.checkOrFail(err, "clean up successful", "could not cleanup namespace")
}

func (j *janitor) getNs(ctx context.Context, nsName string) (nsId string, err error) {
	nsList, err := j.sdApi.ListNamespaces(ctx)
	if err != nil {
		return "", err
	}

	for _, ns := range nsList {
		if ns.Name == nsName {
			return ns.Id, nil
		}
	}
	return "", errors.New("namespace not found")
}

func (j *janitor) deregInsts(ctx context.Context, svcId string) {
	opColl := cloudmap.NewOperationCollector()
	pages := sd.NewListInstancesPaginator(j.awsSdk, &sd.ListInstancesInput{ServiceId: &svcId})
	for pages.HasMorePages() {
		output, instErr := pages.NextPage(ctx)
		j.checkOrFail(instErr,
			fmt.Sprintf("service has %d instances to clean", len(output.Instances)),
			"could not list instances to cleanup")

		for _, inst := range output.Instances {
			instId := aws.ToString(inst.Id)
			fmt.Printf("found instance to clean: %s\n", instId)
			opColl.Add(func() (opId string, err error) {
				return j.sdApi.DeregisterInstance(ctx, svcId, instId)
			})
		}
	}

	opErr := cloudmap.NewDeregisterInstancePoller(j.sdApi, svcId, opColl.Collect(), opColl.GetStartTime()).Poll(ctx)
	j.checkOrFail(opErr, "instances de-registered", "could not cleanup instances")
}

func (j *janitor) checkOrFail(err error, successMsg string, failMsg string) {
	if err != nil {
		fmt.Printf("%s: %s\n", failMsg, err.Error())
		j.fail()
	}

	if successMsg != "" {
		fmt.Println(successMsg)
	}
}
