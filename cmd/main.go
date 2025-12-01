package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	initConfig()
	Ports := buildRuntimeConfig()
	initListeners(Ports)
	// [TODO] Health Check Service,
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGHUP:
			initConfig()
			Ports := buildRuntimeConfig()
			cleanUnusedBackends()
			initListeners(Ports)
		}
	}

}
