package janitor

import (
	"context"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"os"
)

// CloudMapJanitor handles AWS Cloud Map resource cleanup during integration tests.
type CloudMapJanitor interface {
	// Cleanup removes all instances, services and the namespace from AWS Cloud Map for a given namespace name.
	Cleanup(ctx context.Context, nsName string)
}

type cloudMapJanitor struct {
	sdApi ServiceDiscoveryJanitorApi
	fail  func()
}

// NewDefaultJanitor returns a new janitor object.
func NewDefaultJanitor() CloudMapJanitor {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)

	if err != nil {
		fmt.Printf("unable to configure AWS session: %s", err.Error())
		os.Exit(1)
	}

	return &cloudMapJanitor{
		sdApi: NewServiceDiscoveryJanitorApiFromConfig(&awsCfg),
		fail:  func() { os.Exit(1) },
	}
}

func (j *cloudMapJanitor) Cleanup(ctx context.Context, nsName string) {
	fmt.Printf("Cleaning up all test resources in Cloud Map for namespace : %s\n", nsName)

	nsList, err := j.sdApi.ListNamespaces(ctx)
	j.checkOrFail(err, "", "could not find namespace to clean")

	var nsId string
	for _, ns := range nsList {
		if ns.Name == nsName {
			nsId = ns.Id
		}
	}

	if nsId == "" {
		fmt.Println("namespace does not exist in account, nothing to clean")
		return
	}

	fmt.Printf("found namespace to clean: %s\n", nsId)

	svcs, err := j.sdApi.ListServices(ctx, nsId)
	j.checkOrFail(err,
		fmt.Sprintf("namespace has %d services to clean", len(svcs)),
		"could not find services to clean")

	for _, svc := range svcs {
		fmt.Printf("found service to clean: %s\n", svc.Id)
		j.deregisterInstances(ctx, svc.Id)

		delSvcErr := j.sdApi.DeleteService(ctx, svc.Id)
		j.checkOrFail(delSvcErr, "service deleted", "could not cleanup service")
	}

	opId, err := j.sdApi.DeleteNamespace(ctx, nsId)
	if err == nil {
		fmt.Println("namespace delete in progress")
		_, err = j.sdApi.PollNamespaceOperation(ctx, opId)
	}
	j.checkOrFail(err, "clean up successful", "could not cleanup namespace")
}

func (j *cloudMapJanitor) deregisterInstances(ctx context.Context, svcId string) {
	insts, err := j.sdApi.ListInstances(ctx, svcId)
	j.checkOrFail(err,
		fmt.Sprintf("service has %d instances to clean", len(insts)),
		"could not list instances to cleanup")

	opColl := cloudmap.NewOperationCollector()
	for _, inst := range insts {
		instId := aws.ToString(inst.Id)
		fmt.Printf("found instance to clean: %s\n", instId)
		opColl.Add(func() (opId string, err error) {
			return j.sdApi.DeregisterInstance(ctx, svcId, instId)
		})
	}

	opErr := cloudmap.NewDeregisterInstancePoller(j.sdApi, svcId, opColl.Collect(), opColl.GetStartTime()).Poll(ctx)
	j.checkOrFail(opErr, "instances de-registered", "could not cleanup instances")
}

func (j *cloudMapJanitor) checkOrFail(err error, successMsg string, failMsg string) {
	if err != nil {
		fmt.Printf("%s: %s\n", failMsg, err.Error())
		j.fail()
	}

	if successMsg != "" {
		fmt.Println(successMsg)
	}
}
