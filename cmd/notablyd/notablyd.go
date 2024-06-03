package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"notably/cmd/notablyd/routes"
)

// TODO: Use a config file for things like:
//   The HTTP port on which to start the server.
//   The maximum age (in seconds) of the login cookie.
//   Other things when it becomes necessary (e.g. log settings, HTTPS, etc)

// TODO: Proper level-aware logger, with log file rotation.

var httpPort = "8080"

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile | log.Lmicroseconds | log.LUTC)

	log.Println("Welcome to Notably, a simple backend for a simple multi-user note-taking web service.")

	log.Println("Starting webserver on port ", httpPort)

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGSTOP,
		syscall.SIGHUP,
	)
	defer stop()

	rc := routes.RouterConfig{}
	router := routes.NewRouter(rc)

	// Don't use router.Run(), wrap it in an HTTP server instead.
	// This is so that we can shut it down gracefully.
	// gin.Engine does NOT have a Shutdown() method.
	srvAddr := fmt.Sprintf(":%s", httpPort)
	srv := &http.Server{
		Addr:    srvAddr,
		Handler: router,
	}

	// Initializing the HTTP server in a goroutine so that
	// it won't block the graceful shutdown handling below.
	go func() {
		//router.Run(fmt.Sprintf(":%s", httpPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
