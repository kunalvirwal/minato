package main

import (
	"github.com/kunalvirwal/minato/internal/config"
)

// initConfig loads all the configs from Config.yaml
func initConfig() {
	config.LoadConfig()
}

// buildRuntimeConfig uses the RawConfig to generate servers and loadbalancers
func buildRuntimeConfig() {

}
