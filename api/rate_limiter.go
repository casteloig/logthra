package main

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)


type IDRateLimiter struct {
    tenantIDs map[string]*rate.Limiter
    mu  *sync.RWMutex
    r   rate.Limit
    b   int
}

var (
    // Rate limit = 60req/min
    numberRequestsPerTime = 2
    timeDuration = time.Duration(1)*time.Minute
)

// NewIPRateLimiter .
func NewIPRateLimiter() *IDRateLimiter {
    i := &IDRateLimiter{
        tenantIDs: make(map[string]*rate.Limiter),
        mu:  &sync.RWMutex{},
        r:   rate.Every(timeDuration),
        b:   numberRequestsPerTime,
    }
    
    return i
}

// AddIP creates a new rate limiter and adds it to the ips map,
// using the IP address as the key
func (i *IDRateLimiter) AddTenantID(tenantID string) *rate.Limiter {
    if len(tenantID) != 10 {
        return nil
    } else {
        i.mu.Lock()
        defer i.mu.Unlock()

        limiter := rate.NewLimiter(i.r, i.b)
        i.tenantIDs[tenantID] = limiter

        return limiter
    }
}

// GetLimiter returns the rate limiter for the provided IP address if it exists.
// Otherwise calls AddIP to add IP address to the map
func (i *IDRateLimiter) GetLimiter(tenantID string) *rate.Limiter {
    if len(tenantID) != 10 {
        return nil
    } else {
        i.mu.Lock()
        limiter, exists := i.tenantIDs[tenantID]
        if !exists {
            i.mu.Unlock()
            return i.AddTenantID(tenantID)
        }

        i.mu.Unlock()

        return limiter
    } 
}
