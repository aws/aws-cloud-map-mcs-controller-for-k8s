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

type Service struct {
	Namespace string
	Name      string
	Endpoints []*Endpoint
}

type Endpoint struct {
	Id         string
	IP         string
	Port       int32
	Attributes map[string]string
}

const ipv4Attr = "AWS_INSTANCE_IPV4"
const portAttr = "AWS_INSTANCE_PORT"

// NewEndpointFromInstance converts a Cloud Map InstanceSummary to an endpoint
func NewEndpointFromInstance(inst *types.InstanceSummary) (*Endpoint, error) {
	endpoint := Endpoint{
		Id:         *inst.Id,
		Attributes: make(map[string]string, 0),
	}

	if ipv4, hasIp := inst.Attributes[ipv4Attr]; hasIp {
		endpoint.IP = ipv4
	} else {
		return nil, errors.New(fmt.Sprintf("cannot convert service instance %s to endpoint without IP address", *inst.Id))
	}

	if portStr, hasPort := inst.Attributes[portAttr]; hasPort {
		port, parseError := strconv.Atoi(portStr)

		if parseError != nil {
			return nil, parseError
		}

		endpoint.Port = int32(port)
	} else {
		return nil, errors.New(fmt.Sprintf("cannot convert service instance %s to endpoint without port", *inst.Id))
	}

	for key, val := range inst.Attributes {
		if key != ipv4Attr && key != portAttr {
			endpoint.Attributes[key] = val
		}
	}

	return &endpoint, nil
}

// GetAttributes extracts endpoint attributes for Cloud Map service instance registration
func (e *Endpoint) GetAttributes() map[string]string {
	attrs := make(map[string]string, 0)

	attrs[ipv4Attr] = e.IP

	port := strconv.FormatInt(int64(e.Port), 10)
	attrs[portAttr] = port

	for key, val := range e.Attributes {
		attrs[key] = val
	}

	return attrs
}

func (e *Endpoint) Equals(other *Endpoint) bool {
	return reflect.DeepEqual(e, other)
}

func (e *Endpoint) String() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return ""
	}

	return string(bytes)
}

// EndpointIdFromIPAddress converts an IP address to human readable identifier
func EndpointIdFromIPAddress(address string) string {
	return strings.Replace(address, ".", "_", -1)
}
