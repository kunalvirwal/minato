package state

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/kunalvirwal/minato/internal/balancer"
)

// [TODO] create mutable []backend registry that persists across config reloads

var RuntimeCfg MinatoCfg

// MinatoCfg is a structure to hold the current runtime config
type MinatoCfg struct {

	// Config is an atomic pointer so it can be atomically swapped during hot reload
	Config atomic.Pointer[ConfigHolder]

	// Keeps track of all HTTP servers running
	Lm ListenerManager
}

// ConfigHolder stores the domain to PathHandler mapping.
// This is needed because we can have multiple services which
// only differ by host path and not the domain.
type ConfigHolder struct {
	Router map[string][]*PathHandler
}

// Stores the path prefix and loadbalancer for this service
type PathHandler struct {
	PathPrefix string
	LB         balancer.LoadBalancer
}

// Keeps a track of port to http.Server mapping
type ListenerManager struct {
	servers map[uint64]*http.Server
	mu      sync.Mutex
}
