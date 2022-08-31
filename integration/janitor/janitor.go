package janitor

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// CloudMapJanitor handles AWS Cloud Map resource cleanup during integration tests.
type CloudMapJanitor interface {
	// Cleanup removes all instances, services and the namespace from AWS Cloud Map for a given namespace name.
	Cleanup(ctx context.Context, nsName string)
}

type cloudMapJanitor struct {
	clusterId    string
	clusterSetId string
	sdApi        ServiceDiscoveryJanitorApi
	fail         func()
}

// NewDefaultJanitor returns a new janitor object.
func NewDefaultJanitor(clusterId string, clusterSetId string) CloudMapJanitor {
	awsCfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		fmt.Printf("unable to configure AWS session: %s", err.Error())
		os.Exit(1)
	}

	return &cloudMapJanitor{
		clusterId:    clusterId,
		clusterSetId: clusterSetId,
		sdApi:        NewServiceDiscoveryJanitorApiFromConfig(&awsCfg),
		fail:         func() { os.Exit(1) },
	}
}

func (j *cloudMapJanitor) Cleanup(ctx context.Context, nsName string) {
	fmt.Printf("Cleaning up all test resources in Cloud Map for namespace : %s\n", nsName)

	nsMap, err := j.sdApi.GetNamespaceMap(ctx)
	j.checkOrFail(err, "", "could not find namespace to clean")

	ns, found := nsMap[nsName]
	if !found {
		fmt.Println("namespace does not exist in account, nothing to clean")
		return
	}

	fmt.Printf("found namespace to clean: %s\n", ns.Id)

	svcIdMap, err := j.sdApi.GetServiceIdMap(ctx, ns.Id)
	j.checkOrFail(err,
		fmt.Sprintf("namespace has %d services to clean", len(svcIdMap)),
		"could not find services to clean")

	for svcName, svcId := range svcIdMap {
		fmt.Printf("found service to clean: %s\n", svcId)
		j.deregisterInstances(ctx, nsName, svcName, svcId)

		delSvcErr := j.sdApi.DeleteService(ctx, svcId)
		j.checkOrFail(delSvcErr, "service deleted", "could not cleanup service")
	}

	opId, err := j.sdApi.DeleteNamespace(ctx, ns.Id)
	if err == nil {
		fmt.Println("namespace delete in progress")
		_, err = j.sdApi.PollNamespaceOperation(ctx, opId)
	}
	j.checkOrFail(err, "clean up successful", "could not cleanup namespace")
}

func (j *cloudMapJanitor) deregisterInstances(ctx context.Context, nsName string, svcName string, svcId string) {
	queryParameters := map[string]string{
		model.ClusterSetIdAttr: j.clusterSetId,
	}

	insts, err := j.sdApi.DiscoverInstances(ctx, nsName, svcName, &queryParameters)
	j.checkOrFail(err,
		fmt.Sprintf("service has %d instances to clean", len(insts)),
		"could not list instances to cleanup")

	opColl := cloudmap.NewOperationCollector()
	for _, inst := range insts {
		instId := aws.ToString(inst.InstanceId)
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
