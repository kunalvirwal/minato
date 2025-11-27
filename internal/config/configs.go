package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/kunalvirwal/minato/internal/balancer"
	"github.com/kunalvirwal/minato/internal/types"
	"github.com/kunalvirwal/minato/internal/utils"
	"gopkg.in/yaml.v3"
)

var (
	// Path to the config file
	configFile = "./config.yml"
)

// LoadConfig loads the configurations from the config file
func LoadConfig() {
	f, err := os.Open(configFile)
	if err != nil {
		utils.LogNewError("Unable to read config file: " + err.Error())
	}
	defer f.Close()

	var Cfg types.Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&Cfg)
	if err != nil {
		utils.LogError(err)
		os.Exit(1)
	}
	if err := validateConfig(&Cfg); err != nil {
		utils.LogError(err)
	}
	generateRuntimeConfig(&Cfg)
}

func validateConfig(cfg *types.Config) error {
	// There should be atleast one service defined
	if len(cfg.Services) == 0 {
		return errors.New("No services defined in config file")
	}

	serviceNames := make(map[string]bool)
	for i, service := range cfg.Services {
		// No empty service names
		if service.Name == "" {
			return fmt.Errorf("Service at index %d has no name", i)
		}
		// No duplicate service names
		if serviceNames[service.Name] {
			return fmt.Errorf("Duplicate service name found: %s", service.Name)
		}
		serviceNames[service.Name] = true

		// Validate port
		if service.Port <= 0 || service.Port > 65535 {
			return fmt.Errorf("Invalid port %d in service %s", service.Port, service.Name)
		}

		// Validate balancer type
		if service.Balancer != balancer.Round_robin && service.Balancer != balancer.Least_conn {
			return fmt.Errorf("Invalid balancer type %s in service %s", service.Balancer, service.Name)
		}

		// There should be atleast one host
		if len(service.Hosts) == 0 {
			return fmt.Errorf("No hosts defined for service %s", service.Name)
		}

		// There should be atleast one upstream
		if len(service.Upstreams) == 0 {
			return fmt.Errorf("No upstreams defined for service %s", service.Name)
		}

		upstreamHosts := make(map[string]bool)
		for j, upstream := range service.Upstreams {
			// No empty upstream host
			if upstream.Host == "" {
				return fmt.Errorf("Upstream at index %d in service %s has no host", j, service.Name)
			}

			// validate the upstream URL
			parsed, err := url.Parse(upstream.Host)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				return fmt.Errorf("service '%s': upstream[%d] has invalid host URL '%s'", service.Name, j, upstream.Host)
			}

			// No duplicate upstream hosts
			if upstreamHosts[upstream.Host] {
				return fmt.Errorf("Duplicate upstream host %s found in service %s", upstream.Host, service.Name)
			}
			upstreamHosts[upstream.Host] = true
		}
	}
	return nil
}

func generateRuntimeConfig(Cfg *types.Config) {
	fmt.Println(Cfg)
}
