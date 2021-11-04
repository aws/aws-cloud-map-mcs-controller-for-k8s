package scenarios

import (
	"context"
	"fmt"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/controllers"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"k8s.io/apimachinery/pkg/util/wait"
	"strconv"
	"strings"
	"time"
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

func NewExportServiceScenario(cfg *aws.Config, nsName string, svcName string, portStr string, ips string) (ExportServiceScenario, error) {
	endpts := make([]*model.Endpoint, 0)

	port, parseError := strconv.ParseUint(portStr, 10, 16)
	if parseError != nil {
		return nil, parseError
	}

	for _, ip := range strings.Split(ips, ",") {
		endpts = append(endpts, &model.Endpoint{
			Id:         model.EndpointIdFromIPAddress(ip),
			IP:         ip,
			Port:       int32(port),
			Attributes: make(map[string]string, 0),
		})
	}

	return &exportServiceScenario{
		sdClient: cloudmap.NewServiceDiscoveryClientWithCustomCache(cfg,
			&cloudmap.SdCacheConfig{
				NsTTL:    time.Second,
				SvcTTL:   time.Second,
				EndptTTL: time.Second,
			}),
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
			delete(actual.Attributes, controllers.K8sVersionAttr)
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
