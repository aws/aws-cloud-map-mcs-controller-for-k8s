package controllers

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"strings"

	multiclusterv1alpha1 "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	"github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/model"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// DerivedServiceAnnotation annotates a ServiceImport with derived Service name
	DerivedServiceAnnotation = "multicluster.k8s.aws/derived-service"

	// ServiceExportFinalizer finalizer to perform cloudmap resource cleanup on delete
	ServiceExportFinalizer = "multicluster.k8s.aws/service-export-finalizer"

	// LabelServiceImportName indicates the name of the multi-cluster service that an EndpointSlice belongs to.
	LabelServiceImportName = "multicluster.kubernetes.io/service-name"

	// LabelDerivedServiceOriginatingName indicates the name of the multi-cluster service that the derived service originated from.
	LabelDerivedServiceOriginatingName = "multicluster.kubernetes.io/service-name"

	// LabelEndpointSliceManagedBy indicates the name of the entity that manages the EndpointSlice.
	LabelEndpointSliceManagedBy = "endpointslice.kubernetes.io/managed-by"

	// LabelSourceCluster indicates the id of the cluster the object was created for
	LabelSourceCluster = "multicluster.kubernetes.io/source-cluster"

	// ValueEndpointSliceManagedBy indicates the name of the entity that manages the EndpointSlice.
	ValueEndpointSliceManagedBy = "aws-cloud-map-mcs-controller-for-k8s"
)

func AddressTypeToIPType(addressType discovery.AddressType) (model.IPType, error) {
	switch addressType {
	case discovery.AddressTypeIPv6:
		return model.IPV6Type, nil
	case discovery.AddressTypeIPv4:
		return model.IPV4Type, nil
	default:
		return "", fmt.Errorf("unsupported address type %s", addressType)
	}
}

func IPTypeToAddressType(ipType model.IPType) discovery.AddressType {
	switch ipType {
	case model.IPV6Type:
		return discovery.AddressTypeIPv6
	case model.IPV4Type:
		return discovery.AddressTypeIPv4
	default:
		return discovery.AddressTypeIPv4
	}
}

// ServicePortToPort converts a k8s service port to internal model port
func ServicePortToPort(svcPort v1.ServicePort) model.Port {
	return model.Port{
		Name:       svcPort.Name,
		Port:       svcPort.Port,
		TargetPort: svcPort.TargetPort.String(),
		Protocol:   string(svcPort.Protocol),
	}
}

// ServiceImportPortToPort converts a service import port to an internal model port
func ServiceImportPortToPort(svcPort multiclusterv1alpha1.ServicePort) model.Port {
	return model.Port{
		Name:     svcPort.Name,
		Port:     svcPort.Port,
		Protocol: string(svcPort.Protocol),
	}
}

// EndpointPortToPort converts a k8s endpoint port to an internal model port
func EndpointPortToPort(port discovery.EndpointPort) model.Port {
	return model.Port{
		Name:     *port.Name,
		Port:     *port.Port,
		Protocol: string(*port.Protocol),
	}
}

// PortToServicePort converts an internal model port to a k8s service port
func PortToServicePort(port model.Port) v1.ServicePort {
	return v1.ServicePort{
		Name:       port.Name,
		Protocol:   v1.Protocol(port.Protocol),
		Port:       port.Port,
		TargetPort: intstr.Parse(port.TargetPort),
	}
}

// PortToServiceImportPort converts an internal model port to a service import port
func PortToServiceImportPort(port model.Port) multiclusterv1alpha1.ServicePort {
	return multiclusterv1alpha1.ServicePort{
		Name:     port.Name,
		Protocol: v1.Protocol(port.Protocol),
		Port:     port.Port,
	}
}

// PortToEndpointPort converts an internal model port to a k8s endpoint port
func PortToEndpointPort(port model.Port) discovery.EndpointPort {
	protocol := v1.Protocol(port.Protocol)
	return discovery.EndpointPort{
		Name:     &port.Name,
		Protocol: &protocol,
		Port:     &port.Port,
	}
}

// ExtractServicePorts extracts all unique service ports from a slice of endpoints
func ExtractServicePorts(endpoints []*model.Endpoint) (servicePorts []*model.Port) {
	uniquePorts := make(map[string]model.Port)
	for _, ep := range endpoints {
		uniquePorts[ep.ServicePort.GetID()] = ep.ServicePort
	}
	for _, servicePort := range uniquePorts {
		portRef := servicePort
		servicePorts = append(servicePorts, &portRef)
	}
	return servicePorts
}

// ExtractEndpointPorts extracts all unique endpoint ports from a slice of endpoints
func ExtractEndpointPorts(endpoints []*model.Endpoint) (endpointPorts []*model.Port) {
	uniquePorts := make(map[string]model.Port)
	for _, ep := range endpoints {
		uniquePorts[ep.EndpointPort.GetID()] = ep.EndpointPort
	}
	for _, endpointPort := range uniquePorts {
		portRef := endpointPort
		endpointPorts = append(endpointPorts, &portRef)
	}
	return endpointPorts
}

func PortsEqualIgnoreOrder(a, b []*model.Port) (equal bool) {
	idsA := make([]string, len(a))
	idsB := make([]string, len(b))
	for i, port := range a {
		idsA[i] = port.GetID()
	}
	for i, port := range b {
		idsB[i] = port.GetID()
	}
	less := func(x, y string) bool { return x < y }
	equalIgnoreOrder := cmp.Diff(idsA, idsB, cmpopts.SortSlices(less)) == ""
	return equalIgnoreOrder
}

func IPsEqualIgnoreOrder(a, b []string) (equal bool) {
	less := func(x, y string) bool { return x < y }
	equalIgnoreOrder := cmp.Diff(a, b, cmpopts.SortSlices(less)) == ""
	return equalIgnoreOrder
}

// GetClusterIpsFromServices returns list of ClusterIPs from services
func GetClusterIpsFromServices(services []*v1.Service) []string {
	clusterIPs := make([]string, 0)
	for _, svc := range services {
		clusterIPs = append(clusterIPs, svc.Spec.ClusterIP)
	}
	return clusterIPs
}

// DerivedName computes the "placeholder" name for an imported service
func DerivedName(namespace string, name string, clusterId string) string {
	hash := sha256.New()
	hash.Write([]byte(namespace + name + clusterId))
	return "imported-" + strings.ToLower(base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(hash.Sum(nil)))[:10]
}

// CreateServiceImportStruct creates struct representation of a ServiceImport
func CreateServiceImportStruct(svc *model.Service, clusterIds []string, servicePorts []*model.Port) *multiclusterv1alpha1.ServiceImport {
	serviceImportPorts := make([]multiclusterv1alpha1.ServicePort, 0)
	for _, port := range servicePorts {
		serviceImportPorts = append(serviceImportPorts, PortToServiceImportPort(*port))
	}

	clusters := make([]multiclusterv1alpha1.ClusterStatus, 0)
	for _, clusterId := range clusterIds {
		clusters = append(clusters, multiclusterv1alpha1.ClusterStatus{
			Cluster: clusterId,
		})
	}

	return &multiclusterv1alpha1.ServiceImport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: svc.Namespace,
			Name:      svc.Name,
			Annotations: map[string]string{
				DerivedServiceAnnotation: CreateDerivedServiceAnnotation(svc.Namespace, svc.Name, clusterIds),
			},
		},
		Spec: multiclusterv1alpha1.ServiceImportSpec{
			IPs:   []string{},
			Type:  ServiceTypetoServiceImportType(svc.Endpoints[0].ServiceType),
			Ports: serviceImportPorts,
		},
		Status: multiclusterv1alpha1.ServiceImportStatus{
			Clusters: clusters,
		},
	}
}

// CreateDerivedServiceStruct creates struct representation of a derived service
func CreateDerivedServiceStruct(svcImport *multiclusterv1alpha1.ServiceImport, importedSvcPorts []*model.Port, clusterId string) *v1.Service {
	ownerRef := metav1.NewControllerRef(svcImport, schema.GroupVersionKind{
		Version: svcImport.TypeMeta.APIVersion,
		Kind:    svcImport.TypeMeta.Kind,
	})

	svcPorts := make([]v1.ServicePort, 0)
	for _, svcPort := range importedSvcPorts {
		svcPorts = append(svcPorts, PortToServicePort(*svcPort))
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				LabelSourceCluster:                 clusterId,
				LabelDerivedServiceOriginatingName: svcImport.Name,
			},
			Namespace:       svcImport.Namespace,
			Name:            DerivedName(svcImport.Namespace, svcImport.Name, clusterId),
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
		Spec: v1.ServiceSpec{
			Type:  v1.ServiceTypeClusterIP,
			Ports: svcPorts,
		},
	}

	// if svcImport is Headless type, specify ClusterIP field to "None"
	if svcImport.Spec.Type == multiclusterv1alpha1.Headless {
		svc.Spec.ClusterIP = "None"
	}

	return svc
}

func CreateEndpointForSlice(svc *v1.Service, endpoint *model.Endpoint) discovery.Endpoint {
	ep := discovery.Endpoint{
		TargetRef: &v1.ObjectReference{
			Kind:            "Service",
			Namespace:       svc.Namespace,
			Name:            svc.Name,
			UID:             svc.ObjectMeta.UID,
			ResourceVersion: svc.ObjectMeta.ResourceVersion,
		},
		Addresses: []string{endpoint.IP},
		Conditions: discovery.EndpointConditions{
			Ready: &endpoint.Ready,
		},
	}
	if endpoint.Hostname != "" {
		ep.Hostname = &endpoint.Hostname
	}
	if endpoint.Nodename != "" {
		ep.NodeName = &endpoint.Nodename
	}
	return ep
}

func CreateEndpointSliceStruct(svc *v1.Service, svcImportName string, clusterId string, ipType model.IPType) *discovery.EndpointSlice {
	return &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				// derived Service name
				discovery.LabelServiceName: svc.Name,
				// original ServiceImport name
				LabelServiceImportName: svcImportName,
				// 'managed-by' label set to controller
				LabelEndpointSliceManagedBy: ValueEndpointSliceManagedBy,
				// 'source-cluster' label set to current cluster
				LabelSourceCluster: clusterId,
			},
			GenerateName: svc.Name + "-",
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(svc, schema.GroupVersionKind{
				Version: svc.TypeMeta.APIVersion,
				Kind:    svc.TypeMeta.Kind,
			})},
			Namespace: svc.Namespace,
		},
		AddressType: IPTypeToAddressType(ipType),
	}
}

// ExtractServiceType finds the ServiceType of a given service as Headless/ClusterSetIP
func ExtractServiceType(svc *v1.Service) model.ServiceType {
	if svc.Spec.ClusterIP == "None" {
		return model.HeadlessType
	}
	return model.ClusterSetIPType
}

// CreateDerivedServiceAnnotation creates a JSON object containing a slice of maps of clusterIds and derived service names
func CreateDerivedServiceAnnotation(namespace string, name string, clusterIds []string) string {
	clusters := make([]map[string]string, 0, len(clusterIds))
	for _, clusterId := range clusterIds {
		clusters = append(clusters, map[string]string{
			"cluster":         clusterId,
			"derived-service": DerivedName(namespace, name, clusterId),
		})
	}
	// create JSON
	jsonBytes, err := json.Marshal(clusters)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

// ServiceTypetoServiceImportType converts model service type to multicluster ServiceImport type
func ServiceTypetoServiceImportType(serviceType model.ServiceType) multiclusterv1alpha1.ServiceImportType {
	if serviceType == model.HeadlessType {
		return multiclusterv1alpha1.Headless
	}

	return multiclusterv1alpha1.ClusterSetIP
}
