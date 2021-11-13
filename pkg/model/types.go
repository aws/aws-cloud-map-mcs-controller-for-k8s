package model

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"reflect"
	"strconv"
)

// Resource encapsulates a ID/name pair.
type Resource struct {
	Id   string
	Name string
}

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

// Endpoint holds basic values and attributes for an endpoint.
type Endpoint struct {
	Id           string
	IP           string
	EndpointPort Port
	ServicePort  Port
	Attributes   map[string]string
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
	EndpointIpv4Attr      = "AWS_INSTANCE_IPV4"
	EndpointPortAttr      = "AWS_INSTANCE_PORT"
	EndpointPortNameAttr  = "ENDPOINT_PORT_NAME"
	EndpointProtocolAttr  = "ENDPOINT_PROTOCOL"
	ServicePortNameAttr   = "SERVICE_PORT_NAME"
	ServicePortAttr       = "SERVICE_PORT"
	ServiceTargetPortAttr = "SERVICE_TARGET_PORT"
	ServiceProtocolAttr   = "SERVICE_PROTOCOL"
)

// NewEndpointFromInstance converts a Cloud Map HttpInstanceSummary to an endpoint.
func NewEndpointFromInstance(inst *types.HttpInstanceSummary) (endpointPtr *Endpoint, err error) {
	endpoint := Endpoint{
		Id:         *inst.InstanceId,
		Attributes: make(map[string]string),
	}
	attributes := make(map[string]string)
	for key, value := range inst.Attributes {
		attributes[key] = value
	}

	// Remove and set the IP, Port, Port
	if endpoint.IP, err = removeStringAttr(attributes, EndpointIpv4Attr); err != nil {
		return nil, err
	}

	if endpoint.EndpointPort, err = endpointPortFromAttr(attributes); err != nil {
		return nil, err
	}

	if endpoint.ServicePort, err = servicePortFromAttr(attributes); err != nil {
		return nil, err
	}

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
			return 0, fmt.Errorf("failed to parse the %s as int with error %s",
				attr, parseError.Error())
		}
		delete(attributes, attr)
		return int32(parsedValue), nil
	}
	return 0, fmt.Errorf("cannot find the attribute %s", attr)
}

// GetCloudMapAttributes extracts endpoint attributes for Cloud Map service instance registration.
func (e *Endpoint) GetCloudMapAttributes() map[string]string {
	attrs := make(map[string]string)

	attrs[EndpointIpv4Attr] = e.IP
	attrs[EndpointPortAttr] = strconv.Itoa(int(e.EndpointPort.Port))
	attrs[EndpointProtocolAttr] = e.EndpointPort.Protocol
	attrs[EndpointPortNameAttr] = e.EndpointPort.Name
	attrs[ServicePortNameAttr] = e.ServicePort.Name
	attrs[ServicePortAttr] = strconv.Itoa(int(e.ServicePort.Port))
	attrs[ServiceTargetPortAttr] = e.ServicePort.TargetPort
	attrs[ServiceProtocolAttr] = e.ServicePort.Protocol

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

// EndpointIdFromIPAddress converts an IP address to human-readable identifier.
func EndpointIdFromIPAddress(address string, port int32) string {
	return fmt.Sprintf("%s:%d", address, port)
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

func (namespaceType *NamespaceType) IsUnsupported() bool {
	return *namespaceType == UnsupportedNamespaceType
}
