package main

func main() {
	initConfig()
	Ports := buildRuntimeConfig()
	initListeners(Ports)
	// [TODO] Health Check Service, Sighup reload
	<-make(chan struct{})
}
