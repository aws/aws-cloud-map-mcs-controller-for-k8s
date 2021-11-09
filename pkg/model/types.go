package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"reflect"
	"strconv"
	"strings"
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
	Id         string
	IP         string
	Port       int32
	Attributes map[string]string
}

const (
	Ipv4Attr = "AWS_INSTANCE_IPV4"
	PortAttr = "AWS_INSTANCE_PORT"
)

// NewEndpointFromInstance converts a Cloud Map HttpInstanceSummary to an endpoint.
func NewEndpointFromInstance(inst *types.HttpInstanceSummary) (*Endpoint, error) {
	endpoint := Endpoint{
		Id:         *inst.InstanceId,
		Attributes: make(map[string]string, 0),
	}

	if ipv4, hasIp := inst.Attributes[Ipv4Attr]; hasIp {
		endpoint.IP = ipv4
	} else {
		return nil, errors.New(fmt.Sprintf("cannot convert service instance %s to endpoint without IP address",
			*inst.InstanceId))
	}

	if portStr, hasPort := inst.Attributes[PortAttr]; hasPort {
		port, parseError := strconv.ParseUint(portStr, 10, 16)

		if parseError != nil {
			return nil, parseError
		}

		endpoint.Port = int32(port)
	} else {
		return nil, errors.New(fmt.Sprintf("cannot convert service instance %s to endpoint without port",
			*inst.InstanceId))
	}

	for key, val := range inst.Attributes {
		if key != Ipv4Attr && key != PortAttr {
			endpoint.Attributes[key] = val
		}
	}

	return &endpoint, nil
}

// GetCloudMapAttributes extracts endpoint attributes for Cloud Map service instance registration.
func (e *Endpoint) GetCloudMapAttributes() map[string]string {
	attrs := make(map[string]string, 0)

	attrs[Ipv4Attr] = e.IP

	port := strconv.FormatInt(int64(e.Port), 10)
	attrs[PortAttr] = port

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

// EndpointIdFromIPAddress converts an IP address to human readable identifier.
func EndpointIdFromIPAddress(address string) string {
	return strings.Replace(address, ".", "_", -1)
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
