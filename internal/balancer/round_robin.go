package balancer

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/kunalvirwal/minato/internal/backend"
	"github.com/kunalvirwal/minato/internal/utils"
)

type RRbalancer struct {
	SvcName         string
	Port            uint64
	RoundRobinCount atomic.Uint64
	Backends        []*backend.Backend
}

func (lb *RRbalancer) GetPort() uint64 {
	return lb.Port
}

func (lb *RRbalancer) GetAlgorythm() string {
	return Round_robin
}

// Returns the next healthy backend according to Round Robin
func (lb *RRbalancer) GetNextBackend() *backend.Backend {
	n := uint64(len(lb.Backends))
	start := lb.RoundRobinCount.Add(1)
	for i := range n {
		idx := (start + i) % n
		upstream := lb.Backends[idx]
		if upstream.IsAlive() {
			return upstream
		}
	}
	return nil // no healthy backend found
}

func (lb *RRbalancer) ServeProxy(w http.ResponseWriter, r *http.Request) {
	backend := lb.GetNextBackend()
	if backend == nil {
		http.Error(w, "Service Unavailable: No healthy servers available", http.StatusServiceUnavailable)
		utils.LogNewError(fmt.Sprintf("Request Dropped %v: No healthy servers available", lb.SvcName))
		return
	}
	backend.IncrementConnections()
	defer backend.DecrementConnections()
	utils.LogInfo(fmt.Sprintf("Request forwarded to: %v", backend.Address()))
	backend.Serve(w, r)
}

func (lb *RRbalancer) SetBackends(backends []*backend.Backend) {
	lb.Backends = backends
}

func (lb *RRbalancer) GetBackends() []*backend.Backend {
	return lb.Backends
}
