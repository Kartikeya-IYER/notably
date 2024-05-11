package main

import (
	"fmt"
	"log"

	"notably/cmd/notablyd/routes"
)

// TODO: Use a config file for things like:
//   The HTTP port on which to start the server.
//   The maximum age (in seconds) of the login cookie.
//   Other things when it becomes necessary (e.g. log settings, HTTPS, etc)

// TODO: Proper level-aware logger, with log file rotation.

var httpPort = "8080"

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds | log.LUTC)

	log.Println("Welcome to Notably, a simple backend for a simple multi-user note-taking web service.")

	log.Println("Starting webserver on port ", httpPort)

	rc := routes.RouterConfig{}
	router := routes.NewRouter(rc)
	router.Run(fmt.Sprintf(":%s", httpPort))
}
