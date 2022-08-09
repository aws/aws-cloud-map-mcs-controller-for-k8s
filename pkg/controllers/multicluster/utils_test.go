package controllers

import (
	"reflect"
	"strconv"
	"testing"

	multiclusterv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestServicePortToPort(t *testing.T) {
	type args struct {
		svcPort v1.ServicePort
	}
	tests := []struct {
		name string
		args args
		want model.Port
	}{
		{
			name: "happy case",
			args: args{
				svcPort: v1.ServicePort{
					Name:     "http",
					Protocol: v1.ProtocolTCP,
					Port:     80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
					},
				},
			},
			want: model.Port{
				Name:       "http",
				Port:       80,
				TargetPort: "8080",
				Protocol:   "TCP",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServicePortToPort(tt.args.svcPort); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServicePortToPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceImportPortToPort(t *testing.T) {
	type args struct {
		svcImportPort multiclusterv1alpha1.ServicePort
	}
	tests := []struct {
		name string
		args args
		want model.Port
	}{
		{
			name: "happy case",
			args: args{
				svcImportPort: multiclusterv1alpha1.ServicePort{
					Name:     test.PortName1,
					Protocol: v1.ProtocolTCP,
					Port:     80,
				},
			},
			want: model.Port{
				Name:     test.PortName1,
				Port:     80,
				Protocol: test.Protocol1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServiceImportPortToPort(tt.args.svcImportPort); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServiceImportPortToPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpointPortToPort(t *testing.T) {
	type args struct {
		port v1beta1.EndpointPort
	}
	name := "http"
	protocolTCP := v1.ProtocolTCP
	port := int32(80)
	tests := []struct {
		name string
		args args
		want model.Port
	}{
		{
			name: "happy case",
			args: args{
				port: v1beta1.EndpointPort{
					Name:     &name,
					Protocol: &protocolTCP,
					Port:     &port,
				},
			},
			want: model.Port{
				Name:       "http",
				Port:       80,
				TargetPort: "",
				Protocol:   "TCP",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EndpointPortToPort(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EndpointPortToPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortToServicePort(t *testing.T) {
	type args struct {
		port model.Port
	}
	tests := []struct {
		name string
		args args
		want v1.ServicePort
	}{
		{
			name: "happy case",
			args: args{
				port: model.Port{
					Name:       "http",
					Port:       80,
					TargetPort: "8080",
					Protocol:   "TCP",
				},
			},
			want: v1.ServicePort{
				Name:     "http",
				Protocol: v1.ProtocolTCP,
				Port:     80,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8080,
				},
			},
		},
		{
			name: "happy case for string targertPort",
			args: args{
				port: model.Port{
					Name:       "http",
					Port:       80,
					TargetPort: "https",
					Protocol:   "TCP",
				},
			},
			want: v1.ServicePort{
				Name:     "http",
				Protocol: v1.ProtocolTCP,
				Port:     80,
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "https",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PortToServicePort(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PortToServicePort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortToServiceImportPort(t *testing.T) {
	type args struct {
		port model.Port
	}
	tests := []struct {
		name string
		args args
		want multiclusterv1alpha1.ServicePort
	}{
		{
			name: "happy case",
			args: args{
				port: model.Port{
					Name:       test.PortName1,
					Port:       test.Port1,
					TargetPort: test.PortStr2, // ignored
					Protocol:   test.Protocol1,
				},
			},
			want: multiclusterv1alpha1.ServicePort{
				Name:     test.PortName1,
				Protocol: v1.ProtocolTCP,
				Port:     test.Port1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PortToServiceImportPort(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PortToServiceImportPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortToEndpointPort(t *testing.T) {
	name := "http"
	protocolTCP := v1.ProtocolTCP
	port := int32(80)
	type args struct {
		port model.Port
	}
	tests := []struct {
		name string
		args args
		want v1beta1.EndpointPort
	}{
		{
			name: "happy case",
			args: args{
				port: model.Port{
					Name:     "http",
					Port:     80,
					Protocol: "TCP",
				},
			},
			want: v1beta1.EndpointPort{
				Name:     &name,
				Protocol: &protocolTCP,
				Port:     &port,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PortToEndpointPort(tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PortToEndpointPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractServicePorts(t *testing.T) {
	type args struct {
		endpoints []*model.Endpoint
	}
	tests := []struct {
		name string
		args args
		want []*model.Port
	}{
		{
			name: "unique service ports extracted",
			args: args{
				endpoints: []*model.Endpoint{
					{
						ServicePort: model.Port{Protocol: test.Protocol1, Port: test.Port1},
					},
					{
						ServicePort: model.Port{Protocol: test.Protocol1, Port: test.Port2},
					},
					{
						ServicePort: model.Port{Protocol: test.Protocol2, Port: test.Port2},
					},
				},
			},
			want: []*model.Port{
				{Protocol: test.Protocol1, Port: test.Port1},
				{Protocol: test.Protocol1, Port: test.Port2},
				{Protocol: test.Protocol2, Port: test.Port2},
			},
		},
		{
			name: "duplicate and endpoint ports ignored",
			args: args{
				endpoints: []*model.Endpoint{
					{
						ServicePort:  model.Port{Protocol: test.Protocol1, Port: test.Port1},
						EndpointPort: model.Port{Protocol: test.Protocol1, Port: test.Port1},
					},
					{
						ServicePort:  model.Port{Protocol: test.Protocol1, Port: test.Port1},
						EndpointPort: model.Port{Protocol: test.Protocol2, Port: test.Port2},
					},
				},
			},
			want: []*model.Port{
				{Protocol: test.Protocol1, Port: test.Port1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractServicePorts(tt.args.endpoints); !PortsEqualIgnoreOrder(got, tt.want) {
				t.Errorf("ServicePortToPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractEndpointPorts(t *testing.T) {
	type args struct {
		endpoints []*model.Endpoint
	}
	tests := []struct {
		name string
		args args
		want []*model.Port
	}{
		{
			name: "unique endpoint ports extracted",
			args: args{
				endpoints: []*model.Endpoint{
					{
						EndpointPort: model.Port{Protocol: test.Protocol1, Port: test.Port1},
					},
					{
						EndpointPort: model.Port{Protocol: test.Protocol1, Port: test.Port2},
					},
					{
						EndpointPort: model.Port{Protocol: test.Protocol2, Port: test.Port2},
					},
				},
			},
			want: []*model.Port{
				{Protocol: test.Protocol1, Port: test.Port1},
				{Protocol: test.Protocol1, Port: test.Port2},
				{Protocol: test.Protocol2, Port: test.Port2},
			},
		},
		{
			name: "duplicate and service ports ignored",
			args: args{
				endpoints: []*model.Endpoint{
					{
						EndpointPort: model.Port{Protocol: test.Protocol1, Port: test.Port1},
						ServicePort:  model.Port{Protocol: test.Protocol1, Port: test.Port1},
					},
					{
						EndpointPort: model.Port{Protocol: test.Protocol1, Port: test.Port1},
						ServicePort:  model.Port{Protocol: test.Protocol2, Port: test.Port2},
					},
				},
			},
			want: []*model.Port{
				{Protocol: test.Protocol1, Port: test.Port1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractEndpointPorts(tt.args.endpoints); !PortsEqualIgnoreOrder(got, tt.want) {
				t.Errorf("ServicePortToPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortsEqualIgnoreOrder(t *testing.T) {
	type args struct {
		portsA []*model.Port
		portsB []*model.Port
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ports equal same order",
			args: args{
				portsA: []*model.Port{
					{Protocol: test.Protocol1, Port: test.Port1},
					{Protocol: test.Protocol2, Port: test.Port2},
				},
				portsB: []*model.Port{
					{Protocol: test.Protocol1, Port: test.Port1},
					{Protocol: test.Protocol2, Port: test.Port2},
				},
			},
			want: true,
		},
		{
			name: "ports equal different order",
			args: args{
				portsA: []*model.Port{
					{Protocol: test.Protocol1, Port: test.Port1},
					{Protocol: test.Protocol2, Port: test.Port2},
				},
				portsB: []*model.Port{
					{Protocol: test.Protocol2, Port: test.Port2},
					{Protocol: test.Protocol1, Port: test.Port1},
				},
			},
			want: true,
		},
		{
			name: "ports not equal",
			args: args{
				portsA: []*model.Port{
					{Protocol: test.Protocol1, Port: test.Port1},
					{Protocol: test.Protocol2, Port: test.Port2},
				},
				portsB: []*model.Port{
					{Protocol: test.Protocol1, Port: test.Port1},
					{Protocol: test.Protocol2, Port: 3},
				},
			},
			want: false,
		},
		{
			name: "protocols not equal",
			args: args{
				portsA: []*model.Port{
					{Protocol: test.Protocol1, Port: test.Port1},
					{Protocol: test.Protocol2, Port: test.Port2},
				},
				portsB: []*model.Port{
					{Protocol: test.Protocol1, Port: test.Port1},
					{Protocol: test.Protocol1, Port: test.Port2},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PortsEqualIgnoreOrder(tt.args.portsA, tt.args.portsB); !(got == tt.want) {
				t.Errorf("PortsEqualIgnoreOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPsEqualIgnoreOrder(t *testing.T) {
	type args struct {
		ipsA []string
		ipsB []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ips equal same order",
			args: args{
				ipsA: []string{
					test.ClusterIp1,
					test.ClusterIp2,
				},
				ipsB: []string{
					test.ClusterIp1,
					test.ClusterIp2,
				},
			},
			want: true,
		},
		{
			name: "ips equal different order",
			args: args{
				ipsA: []string{
					test.ClusterIp1,
					test.ClusterIp2,
				},
				ipsB: []string{
					test.ClusterIp2,
					test.ClusterIp1,
				},
			},
			want: true,
		},
		{
			name: "ips not equal",
			args: args{
				ipsA: []string{
					test.ClusterIp1,
					test.ClusterIp2,
				},
				ipsB: []string{
					test.ClusterIp1,
					"10.10.10.3",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IPsEqualIgnoreOrder(tt.args.ipsA, tt.args.ipsB); !(got == tt.want) {
				t.Errorf("IPsEqualIgnoreOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetClusterIpsFromServices(t *testing.T) {
	type args struct {
		services []*v1.Service
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "happy case",
			args: args{
				services: []*v1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: v1.ServiceSpec{
							Type:      v1.ServiceTypeClusterIP,
							ClusterIP: test.ClusterIp1,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: v1.ServiceSpec{
							Type:      v1.ServiceTypeClusterIP,
							ClusterIP: test.ClusterIp2,
						},
					},
				}},
			want: []string{
				test.ClusterIp1, test.ClusterIp2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetClusterIpsFromServices(tt.args.services); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClusterIpsFromServices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDerivedService(t *testing.T) {
	const numTests = 100
	derivedServiceMap := make(map[string]bool)
	for i := 0; i < numTests; i++ {
		namespace := test.HttpNsName
		name := "test-svcname-" + strconv.Itoa(i)
		clusterId := "test-clusterid-" + strconv.Itoa(i)
		derivedService := DerivedName(namespace, name, clusterId)
		assert.NotContains(t, derivedServiceMap, derivedService, "derived service already exists")
		derivedServiceMap[derivedService] = true
	}
	assert.Equal(t, numTests, len(derivedServiceMap))
	assert.True(t, DerivedName(test.HttpNsName, test.SvcName, test.ClusterId1) != DerivedName(test.HttpNsName, test.SvcName, test.ClusterId2))
}

func TestCreateServiceImportStruct(t *testing.T) {
	type args struct {
		servicePorts []*model.Port
		clusterIds   []string
	}
	tests := []struct {
		name string
		args args
		want multiclusterv1alpha1.ServiceImport
	}{
		{
			name: "happy case",
			args: args{
				clusterIds: []string{test.ClusterId1, test.ClusterId2},
				servicePorts: []*model.Port{
					{Name: test.PortName1, Protocol: test.Protocol1, Port: test.Port1},
					{Name: test.PortName2, Protocol: test.Protocol1, Port: test.Port2},
				},
			},
			want: multiclusterv1alpha1.ServiceImport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: test.HttpNsName,
					Name:      test.SvcName,
					Annotations: map[string]string{
						DerivedServiceAnnotation: CreateDerivedServiceAnnotation(test.HttpNsName, test.SvcName, []string{test.ClusterId1, test.ClusterId2}),
					},
				},
				Spec: multiclusterv1alpha1.ServiceImportSpec{
					IPs:  []string{},
					Type: multiclusterv1alpha1.ClusterSetIP,
					Ports: []multiclusterv1alpha1.ServicePort{
						{Name: test.PortName1, Protocol: v1.ProtocolTCP, Port: test.Port1},
						{Name: test.PortName2, Protocol: v1.ProtocolTCP, Port: test.Port2},
					},
				},
				Status: multiclusterv1alpha1.ServiceImportStatus{
					Clusters: []multiclusterv1alpha1.ClusterStatus{
						{
							Cluster: test.ClusterId1,
						},
						{
							Cluster: test.ClusterId2,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateServiceImportStruct(test.HttpNsName, test.SvcName, tt.args.clusterIds, tt.args.servicePorts); !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("CreateServiceImportStruct() = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestExtractServiceType(t *testing.T) {
	tests := []struct {
		name string
		svc  *v1.Service
		want model.ServiceType
	}{
		{
			name: "cluster ip type",
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      test.SvcName,
					Namespace: test.HttpNsName,
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{{
						Name:       test.PortName1,
						Protocol:   test.Protocol1,
						Port:       test.ServicePort1,
						TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: test.Port1},
					}},
					ClusterIP: "10.108.89.43",
				},
				Status: v1.ServiceStatus{},
			},
			want: model.ClusterSetIPType,
		},
		{
			name: "headless type",
			svc: &v1.Service{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      test.SvcName,
					Namespace: test.HttpNsName,
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{{
						Name:       test.PortName1,
						Protocol:   test.Protocol1,
						Port:       test.ServicePort1,
						TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: test.Port1},
					}},
					ClusterIP: "None",
				},
				Status: v1.ServiceStatus{},
			},
			want: model.HeadlessType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractServiceType(tt.svc); got != tt.want {
				t.Errorf("ExtractServiceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateDerivedServiceAnnotation(t *testing.T) {
	type args struct {
		namespace  string
		name       string
		clusterIds []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "create derived service annotation",
			args: args{
				namespace:  test.HttpNsName,
				name:       test.SvcName,
				clusterIds: []string{test.ClusterId1, test.ClusterId2},
			},
			want: "[{\"cluster\":\"test-mcs-clusterid-1\",\"derived-service\":\"imported-vm6pdvp7di\"},{\"cluster\":\"test-mcs-clusterid-2\",\"derived-service\":\"imported-i8hm9c3um2\"}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateDerivedServiceAnnotation(tt.args.namespace, tt.args.name, tt.args.clusterIds); got != tt.want {
				t.Errorf("CreateDerivedServiceAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}
