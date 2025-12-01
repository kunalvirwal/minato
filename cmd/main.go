package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	err := initConfig()
	if err == nil {
		updateMinato(true)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	for sig := range sigCh {
		if sig == syscall.SIGHUP {
			err := initConfig()
			if err == nil {
				updateMinato(false)
			}
		}
	}

}

func updateMinato(coldstart bool) {
	Ports := buildRuntimeConfig()
	if coldstart {
		startHealthchecks()
	} else {
		cleanUnusedBackends()
	}
	initListeners(Ports)
}
