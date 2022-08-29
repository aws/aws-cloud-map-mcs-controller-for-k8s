package scenarios

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/common"
	multiclustercontrollers "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/controllers/multicluster"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultScenarioPollInterval = 10 * time.Second
	defaultScenarioPollTimeout  = 2 * time.Minute
)

// ExportServiceScenario defines an integration test against a service export to check creation of namespace, service,
// and endpoint export.
type ExportServiceScenario interface {
	// Run executes the service export integration test scenario, returning any error.
	Run() error
}

type exportServiceScenario struct {
	sdClient    cloudmap.ServiceDiscoveryClient
	expectedSvc model.Service
}

func NewExportServiceScenario(cfg *aws.Config, nsName string, svcName string, clusterId string, clusterSetId string, portStr string, servicePortStr string, serviceType string, ips string) (ExportServiceScenario, error) {
	endpts := make([]*model.Endpoint, 0)

	port, parseError := strconv.ParseUint(portStr, 10, 16)
	if parseError != nil {
		return nil, parseError
	}
	servicePort, parseError := strconv.ParseUint(servicePortStr, 10, 16)
	if parseError != nil {
		return nil, parseError
	}

	for _, ip := range strings.Split(ips, ",") {
		endpointPort := model.Port{
			Port:     int32(port),
			Protocol: string(v1.ProtocolTCP),
		}
		endpts = append(endpts, &model.Endpoint{
			Id: model.EndpointIdFromIPAddressAndPort(ip, endpointPort),
			IP: ip,
			ServicePort: model.Port{
				Port:       int32(servicePort),
				TargetPort: portStr,
				Protocol:   string(v1.ProtocolTCP),
			},
			EndpointPort: endpointPort,
			ClusterId:    clusterId,
			ClusterSetId: clusterSetId,
			ServiceType:  model.ServiceType(serviceType),
			Attributes:   make(map[string]string),
		})
	}

	return &exportServiceScenario{
		sdClient: cloudmap.NewServiceDiscoveryClientWithCustomCache(cfg,
			&cloudmap.SdCacheConfig{
				NsTTL:    time.Second,
				SvcTTL:   time.Second,
				EndptTTL: time.Second,
			}, common.NewClusterUtilsForTest(clusterId, clusterSetId)),
		expectedSvc: model.Service{
			Namespace: nsName,
			Name:      svcName,
			Endpoints: endpts,
		},
	}, nil
}

func (e *exportServiceScenario) Run() error {
	fmt.Printf("Seeking expected service: %v\n", e.expectedSvc)

	return wait.Poll(defaultScenarioPollInterval, defaultScenarioPollTimeout, func() (done bool, err error) {
		fmt.Println("Polling service...")
		cmSvc, err := e.sdClient.GetService(context.TODO(), e.expectedSvc.Namespace, e.expectedSvc.Name)
		if err != nil {
			return true, err
		}

		if cmSvc == nil {
			fmt.Println("Service not found.")
			return false, nil
		}

		fmt.Printf("Found service: %v\n", cmSvc)
		return e.compareEndpoints(cmSvc.Endpoints), nil
	})
}

func (e *exportServiceScenario) compareEndpoints(cmEndpoints []*model.Endpoint) bool {
	if len(e.expectedSvc.Endpoints) != len(cmEndpoints) {
		fmt.Println("Endpoints do not match.")
		return false
	}

	for _, expected := range e.expectedSvc.Endpoints {
		match := false
		for _, actual := range cmEndpoints {
			// Ignore K8S instance attribute for the purpose of this test.
			delete(actual.Attributes, multiclustercontrollers.K8sVersionAttr)
			// Ignore SvcExportCreationTimestamp attribute for the purpose of this test by setting value to 0.
			actual.SvcExportCreationTimestamp = 0
			if expected.Equals(actual) {
				match = true
				break
			}
		}
		if !match {
			fmt.Println("Endpoints do not match.")
			return false
		}
	}

	fmt.Println("Endpoints match.")
	return true
}
