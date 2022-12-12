package model

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
)

const (
	HttpNamespaceType       NamespaceType = "HTTP"
	DnsPrivateNamespaceType NamespaceType = "DNS_PRIVATE"
	// UnsupportedNamespaceType Placeholder NamespaceType to denote not supported values
	UnsupportedNamespaceType NamespaceType = ""
)

type NamespaceType string

// Namespace hold namespace attributes
type Namespace struct {
	Id   string
	Name string
	Type NamespaceType
}

// Service holds namespace and endpoint state for a named service.
type Service struct {
	Namespace string
	Name      string
	Endpoints []*Endpoint
}

const (
	HeadlessType     ServiceType = "Headless"
	ClusterSetIPType ServiceType = "ClusterSetIP"
)

type ServiceType string

const (
	IPV4Type IPType = "IPV4Type"
	IPV6Type IPType = "IPV6Type"
)

type IPType string

// Endpoint holds basic values and attributes for an endpoint.
type Endpoint struct {
	Id                             string
	IP                             string
	IPType                         IPType
	EndpointPort                   Port
	ServicePort                    Port
	ClusterId                      string
	ClusterSetId                   string
	ServiceType                    ServiceType
	ServiceExportCreationTimestamp int64
	Ready                          bool
	Hostname                       string
	Nodename                       string
	Attributes                     map[string]string
}

type Port struct {
	Name       string
	Port       int32
	TargetPort string
	Protocol   string // TCP, UDP, SCTP
}

// Cloudmap Instances IP and Port is supposed to be AWS_INSTANCE_IPV4 and AWS_INSTANCE_PORT
// Rest are custom attributes
const (
	EndpointIpv4Attr          = "AWS_INSTANCE_IPV4"
	EndpointIpv6Attr          = "AWS_INSTANCE_IPV6"
	EndpointPortAttr          = "AWS_INSTANCE_PORT"
	EndpointPortNameAttr      = "ENDPOINT_PORT_NAME"
	EndpointProtocolAttr      = "ENDPOINT_PROTOCOL"
	EndpointReadyAttr         = "READY"
	EndpointHostnameAttr      = "HOSTNAME"
	EndpointNodeNameAttr      = "NODENAME"
	ClusterIdAttr             = "CLUSTER_ID"
	ClusterSetIdAttr          = "CLUSTERSET_ID"
	ServicePortNameAttr       = "SERVICE_PORT_NAME"
	ServicePortAttr           = "SERVICE_PORT"
	ServiceTargetPortAttr     = "SERVICE_TARGET_PORT"
	ServiceProtocolAttr       = "SERVICE_PROTOCOL"
	ServiceTypeAttr           = "SERVICE_TYPE"
	ServiceExportCreationAttr = "SERVICE_EXPORT_CREATION_TIMESTAMP"
	K8sVersionAttr            = "K8S_CONTROLLER"
)

// NewEndpointFromInstance converts a Cloud Map HttpInstanceSummary to an endpoint.
func NewEndpointFromInstance(inst *types.HttpInstanceSummary) (*Endpoint, error) {
	endpoint := Endpoint{
		Id:         *inst.InstanceId,
		Attributes: make(map[string]string),
	}
	attributes := make(map[string]string)
	for key, value := range inst.Attributes {
		attributes[key] = value
	}

	// Remove and set the IP, Port, Service Port, ServiceType, ClusterId, ClusterSetId

	// ASSUMPTION: Endpoints have either IPV4 OR IPV6, not both.
	if _, exists := attributes[EndpointIpv6Attr]; exists {
		ip, err := removeStringAttr(attributes, EndpointIpv6Attr)
		if err != nil {
			return nil, err
		}
		endpoint.IP = ip
		endpoint.IPType = IPV6Type
	} else {
		ip, err := removeStringAttr(attributes, EndpointIpv4Attr)
		if err != nil {
			return nil, err
		}
		endpoint.IP = ip
		endpoint.IPType = IPV4Type
	}

	endpointPort, err := endpointPortFromAttr(attributes)
	if err != nil {
		return nil, err
	}
	endpoint.EndpointPort = endpointPort

	servicePort, err := servicePortFromAttr(attributes)
	if err != nil {
		return nil, err
	}
	endpoint.ServicePort = servicePort

	serviceTypeStr, err := removeStringAttr(attributes, ServiceTypeAttr)
	if err != nil {
		return nil, err
	}
	endpoint.ServiceType = ServiceType(serviceTypeStr)

	if endpoint.ClusterId, err = removeStringAttr(attributes, ClusterIdAttr); err != nil {
		return nil, err
	}

	if endpoint.ClusterSetId, err = removeStringAttr(attributes, ClusterSetIdAttr); err != nil {
		return nil, err
	}

	if endpoint.Ready, err = removeBoolAttr(attributes, EndpointReadyAttr); err != nil {
		return nil, err
	}

	if endpoint.ServiceExportCreationTimestamp, err = removeTimestampAttr(attributes, ServiceExportCreationAttr); err != nil {
		return nil, err
	}

	// Hostname and Nodename are Optional attributes
	endpoint.Hostname, _ = removeStringAttr(attributes, EndpointHostnameAttr)
	endpoint.Nodename, _ = removeStringAttr(attributes, EndpointNodeNameAttr)

	// Add the remaining attributes
	endpoint.Attributes = attributes

	return &endpoint, err
}

func endpointPortFromAttr(attributes map[string]string) (port Port, err error) {
	port = Port{}
	if port.Name, err = removeStringAttr(attributes, EndpointPortNameAttr); err != nil {
		return port, err
	}
	if port.Port, err = removeIntAttr(attributes, EndpointPortAttr); err != nil {
		return port, err
	}
	if port.Protocol, err = removeStringAttr(attributes, EndpointProtocolAttr); err != nil {
		return port, err
	}
	return port, err
}

func servicePortFromAttr(attributes map[string]string) (port Port, err error) {
	port = Port{}
	if port.TargetPort, err = removeStringAttr(attributes, ServiceTargetPortAttr); err != nil {
		return port, err
	}
	if port.Name, err = removeStringAttr(attributes, ServicePortNameAttr); err != nil {
		return port, err
	}
	if port.Port, err = removeIntAttr(attributes, ServicePortAttr); err != nil {
		return port, err
	}
	if port.Protocol, err = removeStringAttr(attributes, ServiceProtocolAttr); err != nil {
		return port, err
	}
	return port, err
}

func removeStringAttr(attributes map[string]string, attr string) (string, error) {
	if value, hasValue := attributes[attr]; hasValue {
		delete(attributes, attr)
		return value, nil
	}
	return "", fmt.Errorf("cannot find the attribute %s", attr)
}

func removeIntAttr(attributes map[string]string, attr string) (int32, error) {
	if value, hasValue := attributes[attr]; hasValue {
		parsedValue, parseError := strconv.ParseUint(value, 10, 16)
		if parseError != nil {
			return 0, fmt.Errorf("failed to parse the %s as int with error %s", attr, parseError.Error())
		}
		delete(attributes, attr)
		return int32(parsedValue), nil
	}
	return 0, fmt.Errorf("cannot find the attribute %s", attr)
}

func removeBoolAttr(attributes map[string]string, attr string) (bool, error) {
	if value, hasValue := attributes[attr]; hasValue {
		parsedValue, parseError := strconv.ParseBool(value)
		if parseError != nil {
			return false, fmt.Errorf("failed to parse the %s as bool with error %s", attr, parseError.Error())
		}
		delete(attributes, attr)
		return parsedValue, nil
	}
	return false, fmt.Errorf("cannot find the attribute %s", attr)
}

func removeTimestampAttr(attributes map[string]string, attr string) (int64, error) {
	if value, hasValue := attributes[attr]; hasValue {
		parsedValue, parseError := strconv.ParseInt(value, 10, 64)
		if parseError != nil {
			return 0, fmt.Errorf("failed to parse the %s as int with error %s",
				attr, parseError.Error())
		}
		delete(attributes, attr)
		return parsedValue, nil
	}
	return 0, fmt.Errorf("cannot find the attribute %s", attr)
}

// GetCloudMapAttributes extracts endpoint attributes for Cloud Map service instance registration.
func (e *Endpoint) GetCloudMapAttributes() map[string]string {
	attrs := make(map[string]string)

	if e.IPType == IPV4Type {
		attrs[EndpointIpv4Attr] = e.IP
	} else if e.IPType == IPV6Type {
		attrs[EndpointIpv6Attr] = e.IP
	}

	attrs[ClusterIdAttr] = e.ClusterId
	attrs[ClusterSetIdAttr] = e.ClusterSetId
	attrs[EndpointPortAttr] = strconv.Itoa(int(e.EndpointPort.Port))
	attrs[EndpointProtocolAttr] = e.EndpointPort.Protocol
	attrs[EndpointPortNameAttr] = e.EndpointPort.Name
	attrs[ServicePortNameAttr] = e.ServicePort.Name
	attrs[ServicePortAttr] = strconv.Itoa(int(e.ServicePort.Port))
	attrs[ServiceTargetPortAttr] = e.ServicePort.TargetPort
	attrs[ServiceProtocolAttr] = e.ServicePort.Protocol
	attrs[ServiceTypeAttr] = e.ServiceType.String()
	attrs[ServiceExportCreationAttr] = strconv.FormatInt(e.ServiceExportCreationTimestamp, 10)
	attrs[EndpointReadyAttr] = strconv.FormatBool(e.Ready)
	attrs[EndpointHostnameAttr] = e.Hostname
	attrs[EndpointNodeNameAttr] = e.Nodename

	for key, val := range e.Attributes {
		attrs[key] = val
	}

	return attrs
}

// Equals evaluates if two Endpoints are "deeply equal" (including all fields).
func (e *Endpoint) Equals(other *Endpoint) bool {
	return reflect.DeepEqual(e, other)
}

// String gives a string representation for an endpoint.
func (e *Endpoint) String() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return ""
	}

	return string(bytes)
}

// EndpointIdFromIPAddressAndPort converts an IP address to human-readable identifier.
func EndpointIdFromIPAddressAndPort(address string, port Port) string {
	address = strings.ReplaceAll(address, ".", "_")
	address = strings.ReplaceAll(address, ":", "_")
	return fmt.Sprintf("%s-%s-%d", strings.ToLower(port.Protocol), address, port.Port)
}

// Gives string representation for ServiceType
func (serviceType ServiceType) String() string {
	return string(serviceType)
}

func ConvertNamespaceType(nsType types.NamespaceType) (namespaceType NamespaceType) {
	switch nsType {
	case types.NamespaceTypeDnsPrivate:
		return DnsPrivateNamespaceType
	case types.NamespaceTypeHttp:
		return HttpNamespaceType
	default:
		return UnsupportedNamespaceType
	}
}

func (svc *Service) GetEndpoints(clusterId string) (endpts []*Endpoint) {
	for _, endpt := range svc.Endpoints {
		if endpt.ClusterId == clusterId {
			endpts = append(endpts, endpt)
		}
	}
	return endpts
}

func (namespaceType *NamespaceType) IsUnsupported() bool {
	return *namespaceType == UnsupportedNamespaceType
}

func (p *Port) GetID() string {
	return fmt.Sprintf("%s:%d", p.Protocol, p.Port)
}

// Equals evaluates if two Ports are "deeply equal" (including all fields).
func (p *Port) Equals(other *Port) bool {
	return reflect.DeepEqual(p, other)
}

func GetIPTypeFromString(ipTypeStr string) (IPType, error) {
	switch ipTypeStr {
	case string(IPV4Type):
		return IPV4Type, nil
	case string(IPV6Type):
		return IPV6Type, nil
	default:
		return "", fmt.Errorf("Invalid IPType, could not parse from string: %s", ipTypeStr)
	}
}
