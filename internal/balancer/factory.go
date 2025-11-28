package balancer

import (
	"net/http"

	"github.com/kunalvirwal/minato/internal/backend"
)

const (
	Round_robin = "RoundRobin"
	Least_conn  = "LeastConnections"
)

type LoadBalancer interface {

	// Gets the port on which the load balancer is running
	GetPort() uint64

	// gets the next backend according to the algorythm used
	GetNextBackend() *backend.Backend

	// forwards the request to the next server
	ServeProxy(w http.ResponseWriter, r *http.Request)

	// gets the algorythm being used by that load balancer
	GetAlgorythm() string
}
