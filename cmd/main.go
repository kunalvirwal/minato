package main

func main() {
	initConfig()
	buildRuntimeConfig()
	initListeners()
	// [TODO] Health Check Service, Sighup reload
}
