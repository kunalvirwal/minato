package main

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/kunalvirwal/minato/internal/config"
	"github.com/kunalvirwal/minato/internal/state"
	"github.com/kunalvirwal/minato/internal/utils"
)

// initConfig loads all the configs from Config.yaml
func initConfig() {
	config.LoadConfig()
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
	reqHandler := func(w http.ResponseWriter, r *http.Request) {
		domain := r.Host
		// Loads the latest config
		cfg := state.RuntimeCfg.Config.Load()
		ph := cfg.Router[domain]
		if ph == nil {
			fmt.Println(domain)
			utils.LogNewError("A request with unrecognised domain recieved, please update config.yml file or DNS ")
			http.Error(w, "Service not found", http.StatusNotFound)
			return
		}
		for _, pathHandler := range ph {
			if strings.HasPrefix(r.URL.Path, pathHandler.PathPrefix) {
				pathHandler.LB.ServeProxy(w, r)
				return
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
				Handler: http.HandlerFunc(reqHandler),
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
