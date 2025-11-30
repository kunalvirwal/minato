package state

import (
	"net/url"

	"github.com/kunalvirwal/minato/internal/backend"
	"github.com/kunalvirwal/minato/internal/balancer"
	"github.com/kunalvirwal/minato/internal/types"
	"github.com/kunalvirwal/minato/internal/utils"
)

func GenerateRuntimeResources(Cfg *types.Config) {

	var newConfig = ConfigHolder{
		Router: make(map[string][]*PathHandler),
	}

	for _, svc := range Cfg.Services {

		// create backends for a service
		var backends []*backend.Backend
		for _, upstream := range svc.Upstreams {
			// [TODO] Later replace this nil to backend.state if this backend existed prior to reload
			backend := backend.CreateBackend(upstream.Host, upstream.Health_uri, nil)
			backends = append(backends, backend)
		}

		// [TODO] append the backends slice to backend registry if you need to

		// create loadbalancer for this service
		lb := balancer.CreateLoadBalancer(svc.Name, svc.Balancer, svc.Port, backends)
		if lb == nil {
			utils.LogNewError("Invalid balancing algorythm, nil load balancer recieved")
			return
		}

		// Add the created loadbalancer to the state struct
		for _, link := range svc.Hosts {
			parsed, _ := url.Parse(link)
			newConfig.Router[parsed.Host] = append(newConfig.Router[parsed.Host], &PathHandler{
				PathPrefix: parsed.Path,
				LB:         lb,
			})
		}

	}
	// Atomic Swap
	CommitConfig(&newConfig)
}

// Atomically swaps the config
func CommitConfig(cfg *ConfigHolder) {
	RuntimeCfg.Config.Store(cfg)
}
