package balancer

import (
	"fmt"
	"net/http"

	"github.com/kunalvirwal/minato/internal/backend"
	"github.com/kunalvirwal/minato/internal/cache"
	"github.com/kunalvirwal/minato/internal/utils"
)

type LCbalancer struct {
	SvcName  string
	Port     uint64
	Backends []*backend.Backend
}

func (lb *LCbalancer) GetPort() uint64 {
	return lb.Port
}

func (lb *LCbalancer) GetAlgorythm() string {
	return Least_conn
}

// Returns the next healthy backend according to Round Robin
func (lb *LCbalancer) GetNextBackend() *backend.Backend {
	var selected *backend.Backend
	var minConn int64 = -1
	for _, upstream := range lb.Backends {
		if upstream.IsAlive() {
			activeConns := upstream.ActiveConnections()
			if selected == nil || activeConns < minConn {
				minConn = activeConns
				selected = upstream
			}
		}
	}
	return selected
}

func (lb *LCbalancer) ServeProxy(w http.ResponseWriter, r *http.Request) *cache.Response {
	backend := lb.GetNextBackend()
	if backend == nil {
		http.Error(w, "Service Unavailable: No healthy servers available", http.StatusServiceUnavailable)
		utils.LogNewError(fmt.Sprintf("Request Dropped %v: No healthy servers available", lb.SvcName))
		return nil
	}
	backend.IncrementConnections()
	defer backend.DecrementConnections()
	utils.LogInfo(fmt.Sprintf("Request forwarded to: %v", backend.Address()))
	return backend.Serve(w, r)
}

func (lb *LCbalancer) SetBackends(backends []*backend.Backend) {
	lb.Backends = backends
}

func (lb *LCbalancer) GetBackends() []*backend.Backend {
	return lb.Backends
}
