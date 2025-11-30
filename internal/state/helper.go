package state

import (
	"net/url"

	"github.com/kunalvirwal/minato/internal/backend"
	"github.com/kunalvirwal/minato/internal/balancer"
	"github.com/kunalvirwal/minato/internal/types"
	"github.com/kunalvirwal/minato/internal/utils"
)

func GenerateRuntimeResources(Cfg *types.Config) []uint64 {

	// new config for replacement
	var newConfig = ConfigHolder{
		Router: make(map[RouteKey]balancer.LoadBalancer),
	}

	// ports needed in the new config
	newPorts := []uint64{}

	// iterate over all services defined in config
	for _, svc := range Cfg.Services {

		// create backends for a service
		var backends []*backend.Backend
		for _, upstream := range svc.Upstreams {
			// [TODO] Later replace this nil to backend.state if this backend existed prior to reload
			backend := backend.CreateBackend(upstream.Host, upstream.Health_uri, nil)
			backends = append(backends, backend)
		}

		// [TODO] append the backends slice to backend registry if you need to

		// The ports needed in the latest config
		newPorts = append(newPorts, uint64(svc.Port))

		// create loadbalancer for this service
		lb := balancer.CreateLoadBalancer(svc.Name, svc.Balancer, svc.Port, backends)
		if lb == nil {
			utils.LogNewError("Invalid balancing algorythm, nil load balancer recieved")
			return newPorts
		}

		// Add the created loadbalancer to the state struct
		for _, link := range svc.Hosts {
			parsed, _ := url.Parse(link)
			route := RouteKey{
				Domain:     parsed.Host,
				PathPrefix: parsed.Path,
				Port:       uint64(svc.Port),
			}
			newConfig.Router[route] = lb
		}
	}
	// Atomic Swap
	CommitConfig(&newConfig)
	return newPorts
}

func UpdateRuntimeResources(Cfg *types.Config) []uint64 {
	// // new config for replacement
	// var newConfig = ConfigHolder{
	// 	Router: make(map[string][]*PathHandler),
	// }

	// // ports needed in the new config
	// newPorts := []uint64{}

	// oldConfig := RuntimeCfg.Config.Load()
	// // service updater
	// for _, svc := range Cfg.Services {
	// 	// create backends for a service
	// 	updateService(svc, oldConfig)

	// }
	///////////////Change
	return []uint64{}
}

// func updateService(svc types.Service, oldConfig *ConfigHolder) {
// 	// first we find the old LB for this service which has the same port and balancer type

// }

// Atomically swaps the config
func CommitConfig(cfg *ConfigHolder) {
	RuntimeCfg.Config.Store(cfg)
}
