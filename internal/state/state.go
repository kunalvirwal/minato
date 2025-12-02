package state

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/kunalvirwal/minato/internal/backend"
	"github.com/kunalvirwal/minato/internal/balancer"
	"github.com/kunalvirwal/minato/internal/cache"
)

// [TODO] create mutable []backend registry that persists across config reloads

var RuntimeCfg MinatoCfg = MinatoCfg{
	Config: atomic.Pointer[ConfigHolder]{},
	Lm: ListenerManager{
		Listeners: make(map[uint64]*http.Server),
	},
	BackendRegistry: make(map[BackendKey]*backend.Backend),
}

// MinatoCfg is a structure to hold the current runtime config
type MinatoCfg struct {

	// Config is an atomic pointer so it can be atomically swapped during hot reload
	Config atomic.Pointer[ConfigHolder]

	// Keeps track of all HTTP servers running
	Lm ListenerManager

	// Keeps track of backend states across config reloads
	BackendRegistry map[BackendKey]*backend.Backend

	// RWMutex to protect BackendRegistry
	Mu sync.RWMutex
}

// ConfigHolder stores the domain to PathHandler mapping.
// This is needed because we can have multiple services which
// only differ by host path and not the domain.
type ConfigHolder struct {
	Router map[RouteKey]balancer.LoadBalancer
	Cache  cache.Cache
}

// The combination of a URL and port uniquely identifies a loadbalancer
type RouteKey struct {
	Domain     string
	PathPrefix string
	Port       uint64
}

// The combination of a URL and health check URI uniquely identifies a backend
type BackendKey struct {
	Address    string // Stores "host:port/path"
	Health_uri string
}

// Keeps a track of port to http.Server mapping
type ListenerManager struct {
	Listeners map[uint64]*http.Server
	Mu        sync.Mutex
}
