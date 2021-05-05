package model

import (
	"fmt"
	"reflect"
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

func (e *Endpoint) Equals(other *Endpoint) bool {
	return reflect.DeepEqual(e, other)
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("[%s, %s, %d]", e.Id, e.IP, e.Port)
}

// EndpointIdFromIPAddress converts an IP address to human readable identifier
func EndpointIdFromIPAddress(address string) string {
	return strings.Replace(address, ".", "_", -1)
}
