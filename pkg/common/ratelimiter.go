package common

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"
)

const (
	ListNamespaces     Event = "ListNamespaces"
	ListServices       Event = "ListServices"
	GetOperation       Event = "GetOperation"
	RegisterInstance   Event = "RegisterInstance"
	DeregisterInstance Event = "DeregisterInstance"
)

type Event string

type RateLimiter struct {
	rateLimiters map[Event]*rate.Limiter
}

// NewDefaultRateLimiter returns the rate limiters with the default limits for the AWS CloudMap's API calls
func NewDefaultRateLimiter() RateLimiter {
	return RateLimiter{rateLimiters: map[Event]*rate.Limiter{
		// Below are the default limits for the AWS CloudMap's APIs
		// TODO: make it customizable in the future
		ListNamespaces:     rate.NewLimiter(rate.Limit(1), 5),     // 1 ListNamespaces API calls per second
		ListServices:       rate.NewLimiter(rate.Limit(2), 10),    // 2 ListServices API calls per second
		GetOperation:       rate.NewLimiter(rate.Limit(100), 200), // 100 GetOperation API calls per second
		RegisterInstance:   rate.NewLimiter(rate.Limit(50), 100),  // 50 RegisterInstance API calls per second
		DeregisterInstance: rate.NewLimiter(rate.Limit(50), 100),  // 50 DeregisterInstance API calls per second
	}}
}

// Wait blocks until limit permits an event to happen. It returns an error if the Context is canceled, or the expected wait time exceeds the Context's Deadline.
func (r RateLimiter) Wait(ctx context.Context, event Event) error {
	if limiter, ok := r.rateLimiters[event]; ok {
		return limiter.Wait(ctx)
	}
	return fmt.Errorf("event %s not found in the list of limiters", event)
}
