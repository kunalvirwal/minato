package healthcheck

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/kunalvirwal/minato/internal/backend"
	"github.com/kunalvirwal/minato/internal/state"
	"github.com/kunalvirwal/minato/internal/utils"
)

func StartHealthchecks() {
	ticker := time.NewTicker(10 * time.Second)

	// Initial healthcheck run
	state.RuntimeCfg.Mu.RLock()
	for key, backend := range state.RuntimeCfg.BackendRegistry {
		go runHealthCheck(key, backend)
	}
	state.RuntimeCfg.Mu.RUnlock()

	// Periodic healthcheck runs
	for range ticker.C {
		state.RuntimeCfg.Mu.RLock()
		for key, backend := range state.RuntimeCfg.BackendRegistry {
			go runHealthCheck(key, backend)
		}
		state.RuntimeCfg.Mu.RUnlock()
	}

}

func runHealthCheck(key state.BackendKey, backend *backend.Backend) {
	// Fast TCP check
	TCPconnectionTimeout := 1 * time.Second
	HTTPconnectionTimeout := 3 * time.Second
	health_url := "http://" + backend.Address() + key.Health_uri

	// TCP test happens to host:port, it is a layer 4 protocol so it doesn't need http
	conn, err := net.DialTimeout("tcp", backend.Config.URL.Host, TCPconnectionTimeout)
	if err != nil && backend.IsAlive() {
		// utils.LogCustom(utils.Red, "Healthcheck-test", fmt.Sprintf("TCP Healthcheck failed on %v", key.Address))
		utils.LogCustom(utils.Red, "Healthcheck", fmt.Sprintf("%v went offline", key.Address))
		backend.SetHealth(false)
		return
	}
	conn.Close()

	// HTTP health endpoint test
	client := &http.Client{
		Timeout: HTTPconnectionTimeout,
	}
	res, err := client.Get(health_url)
	if err != nil || res.StatusCode != http.StatusOK {
		// utils.LogCustom(utils.Red, "Healthcheck-test", fmt.Sprintf("HTTP Healthcheck failed on %v", key.Address))
		if backend.IsAlive() {
			backend.SetHealth(false)
			utils.LogCustom(utils.Red, "Healthcheck", fmt.Sprintf("%v failing healthchecks", key.Address))
		}
		return
	}

	defer res.Body.Close()

	if !backend.IsAlive() {
		backend.SetHealth(true)
		utils.LogCustom(utils.Green, "Healthcheck", fmt.Sprintf("%v is now online", key.Address))
	}

}
