package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kunalvirwal/minato/internal/balancer"
	"github.com/kunalvirwal/minato/internal/cache"
	"github.com/kunalvirwal/minato/internal/config"
	"github.com/kunalvirwal/minato/internal/healthcheck"
	"github.com/kunalvirwal/minato/internal/state"
	"github.com/kunalvirwal/minato/internal/utils"
)

// initConfig loads all the configs from Config.yaml
func initConfig() error {
	return config.LoadConfig()
}

// buildRuntimeConfig uses the RawConfig to generate servers and loadbalancers
func buildRuntimeConfig() []uint64 {
	return state.GenerateRuntimeResources(config.RawConfig)
}

// initListener removes old Listeners which are not in latest config and starts new Listeners
func initListeners(newPorts []uint64) {

	state.RuntimeCfg.Lm.Mu.Lock()
	defer state.RuntimeCfg.Lm.Mu.Unlock()

	// stop old Listeners
	for port, listener := range state.RuntimeCfg.Lm.Listeners {
		if !slices.Contains(newPorts, port) {
			delete(state.RuntimeCfg.Lm.Listeners, port)
			if listener != nil {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					listener.Shutdown(ctx)
				}()
			}
		}
	}

	// Request handler Logic
	reqHandler := func(port uint64) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {

			host := r.Host
			reqPath := r.URL.Path

			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}

			// Loads the latest config
			cfg := state.RuntimeCfg.Config.Load()

			// Find the load balancer for this domain  and port with the longest matching path prefix
			var LB balancer.LoadBalancer
			longestPrefix := -1
			for key, lb := range cfg.Router {
				if key.Domain != host || key.Port != port {
					continue
				}
				// If this routekey has a path prefix matching the request path
				if strings.HasPrefix(reqPath, key.PathPrefix) {
					if len(key.PathPrefix) > longestPrefix {
						LB = lb
						longestPrefix = len(key.PathPrefix)
					}
				}
			}

			if LB == nil {
				utils.LogNewError("A request with unrecognised domain or path recieved, please update config.yml file or DNS ")
				http.Error(w, "Service not found", http.StatusNotFound)
				return
			}

			var runtimeCache cache.Cache
			key := ""
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				runtimeCache = state.RuntimeCfg.Config.Load().Cache
				if runtimeCache != nil {
					key = cache.BuildCacheKey(r, int(port))
					if resp, found := runtimeCache.Get(key); found {
						utils.LogCustom(utils.Cyan, "Cache", (fmt.Sprintf("Cache hit for %v", key)))
						writeCachedResponse(w, resp)
						return
					}
				}
			}
			resp := LB.ServeProxy(w, r)

			// Store in cache if applicable
			if runtimeCache != nil && resp != nil && key != "" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				hdr := resp.Header
				noCache := false
				ttl := runtimeCache.GetTTL()
				utils.LogCustom(utils.Cyan, "Cache", fmt.Sprintf("Storing cache with key %v", key))
				if cc := hdr.Get("Cache-Control"); cc != "" {

					parts := strings.Split(cc, ",")
					for i := range parts {
						p := strings.TrimSpace(parts[i])
						if p == "no-store" || p == "no-cache" || p == "private" {
							noCache = true
						}

						if n, found := strings.CutPrefix(p, "max-age="); found {
							n, err := strconv.Atoi(n)
							if err == nil {
								if n == 0 {
									noCache = true
								} else {
									ttl = int64(n)
								}
							}
						}
					}

				}

				if !noCache {
					expiresAt := time.Now().Unix() + ttl
					runtimeCache.Set(key, *resp, expiresAt)
					utils.LogInfo(fmt.Sprintf("Response cached for key %v", key))
				}
			}
		}
	}

	// start new listeners
	for _, port := range newPorts {
		_, exists := state.RuntimeCfg.Lm.Listeners[port]
		// if listener does not exist on this port, create one
		if !exists {
			srv := &http.Server{
				Addr:    fmt.Sprintf(":%d", port),
				Handler: http.HandlerFunc(reqHandler(port)),
			}
			state.RuntimeCfg.Lm.Listeners[port] = srv

			go func(srv *http.Server, port uint64) {
				utils.LogInfo(fmt.Sprintf("Listening on port %v", port))
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					utils.LogNewError(fmt.Sprintf("Error in server running on port %d : %v", port, err))
				}
			}(srv, port)
		}
		// else listener already exists on this port, do nothing
	}
}

func writeCachedResponse(w http.ResponseWriter, resp cache.Response) {
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if len(resp.Body) > 0 {
		_, err := w.Write(resp.Body)
		if err != nil {
			utils.LogNewError("Error writing cached response body: " + err.Error())
		}
	}
}

func cleanUnusedBackends() {
	active := make(map[state.BackendKey]bool)

	for _, lb := range state.RuntimeCfg.Config.Load().Router {
		for _, backend := range lb.GetBackends() {
			key := state.BackendKey{
				Address:    backend.Address(),
				Health_uri: backend.Config.Health_uri,
			}
			active[key] = true
		}
	}

	state.RuntimeCfg.Mu.Lock()
	defer state.RuntimeCfg.Mu.Unlock()

	for key := range state.RuntimeCfg.BackendRegistry {
		if !active[key] {
			delete(state.RuntimeCfg.BackendRegistry, key)
			utils.LogInfo(fmt.Sprintf("Cleaning up unused backend: %v", key.Address))
		}
	}
}

func startHealthchecks() {
	go healthcheck.StartHealthchecks()
}
