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
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)

replace github.com/spf13/viper v1.4.0 => github.com/spf13/viper v1.8.0
