package balancer

import (
	"net/http"

	"github.com/kunalvirwal/minato/internal/backend"
	"github.com/kunalvirwal/minato/internal/cache"
)

const (
	Round_robin = "RoundRobin"
	Least_conn  = "LeastConnections"
)

type LoadBalancer interface {

	// Gets the port on which the load balancer is running
	GetPort() uint64

	// gets the next healthy backend according to the algorythm used
	GetNextBackend() *backend.Backend

	// forwards the request to the next server
	ServeProxy(w http.ResponseWriter, r *http.Request) *cache.Response

	// gets the algorythm being used by that load balancer
	GetAlgorythm() string

	// Gets slice of active Backends
	GetBackends() []*backend.Backend

	// Sets Backend slice
	SetBackends(backends []*backend.Backend)
}

func CreateLoadBalancer(svc string, algo string, port int, backends []*backend.Backend) LoadBalancer {
	if algo == Round_robin {
		return &RRbalancer{
			SvcName:  svc,
			Port:     uint64(port),
			Backends: backends,
		}
	}
	return nil
}
