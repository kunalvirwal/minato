package backend

import (
	"net/http"
	"sync/atomic"
)

type BackendConfig struct {
	Address    string
	Health_uri string
	Proxy      http.HandlerFunc
}

type BackendState struct {
	ActiveConnections atomic.Uint64
	Healthy           atomic.Bool
}

type Backend struct {
	Config *BackendConfig
	State  *BackendState
}

func CreateBackend(URL string, Health_uri string, state *BackendState) *Backend {

	// If no previous state exist for this server then create one
	if state == nil {
		state = &BackendState{}
		state.ActiveConnections.Store(0)
		state.Healthy.Store(true)
	}

	config := &BackendConfig{
		Address:    URL,
		Health_uri: Health_uri,
		Proxy:      func(w http.ResponseWriter, r *http.Request) {}, // [TODO] Replace this with custom proxy implementation
	}

	return &Backend{
		Config: config,
		State:  state,
	}
}

// Address returns the upstream URI of this backend
func (b *Backend) Address() string {
	return b.Config.Address
}

// Serve creates a new proxy request to this upstream backend
func (b *Backend) Serve(w http.ResponseWriter, r *http.Request) {
	b.Config.Proxy(w, r)
}

// IsAlive returns the health status of this backend
func (b *Backend) IsAlive() bool {
	return b.State.Healthy.Load()
}

// ActiveConnections returns the number of Active client connections to this backend
func (b *Backend) ActiveConnections() uint64 {
	return b.State.ActiveConnections.Load()
}

// Increments the number of Active Connections
func (b *Backend) IncrementConnections() {
	b.State.ActiveConnections.Add(1)
}

// Decrements the number of Active Connections
func (b *Backend) DecrementConnections() {
	b.State.ActiveConnections.Add(1)
}

// Sets the health status of this backend
func (b *Backend) SetHealth(health bool) {
	b.State.Healthy.Store(health)
}
