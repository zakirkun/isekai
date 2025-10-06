package loadbalancer

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Strategy represents load balancing strategy
type Strategy string

const (
	RoundRobin Strategy = "round_robin"
	LeastConn  Strategy = "least_conn"
	Random     Strategy = "random"
)

// Backend represents a backend server
type Backend struct {
	URL         string
	Healthy     bool
	Connections int32
	mu          sync.RWMutex
}

// LoadBalancer manages backend servers
type LoadBalancer struct {
	backends []*Backend
	current  uint32
	strategy Strategy
	mu       sync.RWMutex
}

// New creates a new load balancer
func New(strategy Strategy) *LoadBalancer {
	return &LoadBalancer{
		backends: make([]*Backend, 0),
		strategy: strategy,
	}
}

// AddBackend adds a backend server
func (lb *LoadBalancer) AddBackend(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backend := &Backend{
		URL:     url,
		Healthy: true,
	}
	lb.backends = append(lb.backends, backend)
}

// RemoveBackend removes a backend server
func (lb *LoadBalancer) RemoveBackend(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, backend := range lb.backends {
		if backend.URL == url {
			lb.backends = append(lb.backends[:i], lb.backends[i+1:]...)
			return
		}
	}
}

// GetBackend returns the next backend based on strategy
func (lb *LoadBalancer) GetBackend() (*Backend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	switch lb.strategy {
	case RoundRobin:
		return lb.roundRobin(), nil
	case LeastConn:
		return lb.leastConn(), nil
	default:
		return lb.roundRobin(), nil
	}
}

// roundRobin implements round-robin load balancing
func (lb *LoadBalancer) roundRobin() *Backend {
	// Find next healthy backend
	attempts := len(lb.backends)
	for i := 0; i < attempts; i++ {
		idx := atomic.AddUint32(&lb.current, 1) % uint32(len(lb.backends))
		backend := lb.backends[idx]

		backend.mu.RLock()
		healthy := backend.Healthy
		backend.mu.RUnlock()

		if healthy {
			return backend
		}
	}

	// Return first backend if none are healthy
	return lb.backends[0]
}

// leastConn implements least connections load balancing
func (lb *LoadBalancer) leastConn() *Backend {
	var selected *Backend
	minConn := int32(1<<31 - 1)

	for _, backend := range lb.backends {
		backend.mu.RLock()
		healthy := backend.Healthy
		conn := atomic.LoadInt32(&backend.Connections)
		backend.mu.RUnlock()

		if healthy && conn < minConn {
			selected = backend
			minConn = conn
		}
	}

	if selected == nil {
		return lb.backends[0]
	}

	return selected
}

// MarkHealthy marks a backend as healthy
func (lb *LoadBalancer) MarkHealthy(url string, healthy bool) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, backend := range lb.backends {
		if backend.URL == url {
			backend.mu.Lock()
			backend.Healthy = healthy
			backend.mu.Unlock()
			return
		}
	}
}

// IncrementConnections increments connection count for a backend
func (b *Backend) IncrementConnections() {
	atomic.AddInt32(&b.Connections, 1)
}

// DecrementConnections decrements connection count for a backend
func (b *Backend) DecrementConnections() {
	atomic.AddInt32(&b.Connections, -1)
}

// GetAllBackends returns all backends with their status
func (lb *LoadBalancer) GetAllBackends() []map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(lb.backends))
	for _, backend := range lb.backends {
		backend.mu.RLock()
		result = append(result, map[string]interface{}{
			"url":         backend.URL,
			"healthy":     backend.Healthy,
			"connections": atomic.LoadInt32(&backend.Connections),
		})
		backend.mu.RUnlock()
	}

	return result
}
