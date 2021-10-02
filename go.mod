module github.com/aws/aws-cloud-map-mcs-controller-for-k8s

go 1.15

require (
	github.com/aws/aws-sdk-go-v2 v1.8.1
	github.com/aws/aws-sdk-go-v2/config v1.6.1
	github.com/aws/aws-sdk-go-v2/service/servicediscovery v1.7.3
	github.com/go-logr/logr v0.3.0
	github.com/golang/mock v1.6.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.7.2
)
