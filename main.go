package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("main: Started")
	defer log.Println("main: Completed")

	// Api service configuration

	// ReadTimeout: It defines how long you allow a connection to be open
	// during a client sends data or max time required for reading the entire
	// request, includig body

	// WriteTimeout: It is maximum duration before timing out writes of the
	// response.
	api := http.Server{
		Addr:         "0.0.0.0:8000",
		Handler:      http.HandlerFunc(Echo),
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 7 * time.Second,
	}

	// Make a channel to listend for errors coming from the listener. Use a
	// Buffered channel so the goroutine can exit if we do not collect error
	serverErrors := make(chan error, 1)

	// Start the service listening for requests
	// the reason for adding ListenAndServe in goroutine because ListenAndServe
	// is a blocking call and if we want to go one doing more work, like
	// making a second instance if ListenAndServe, then we need a separate
	// goroutine
	go func() {
		log.Printf("main: API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// Make a channel to listen for an interrupt or terminal signal from OS
	// Use a buffered channel, as signal package requires it
	// Note: SIGTERM signal is used by kubernetes instead of os.Interrupt
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("error: listening and serving: %s", err)
	case <-shutdown:
		log.Println("main: Starting shutdown")

		// Added a deadline for request completion
		// we can perfrom any chores in this time e.g clearing memory,
		// resources etc.
		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// SetKeepAlivesEnabled will inform the webserver ti not keeo any
		// existing connections alive which basically gives us the gracefull
		// shutdown behavior
		api.SetKeepAlivesEnabled(false)

		// Asking a listener to shutdown without interrupting any active
		// connections. Shutdown works by first closing all open listeners,
		// then closing idle connections and then waiting indefinitely for
		// connections to return to idle and then shutdown. If the provided
		// context expires before Shutdown is complete, shutdown returns
		// the context error, otherwise it returns any error returned from
		// closign the servers listeners
		err := api.Shutdown(ctx)
		if err != nil {
			log.Printf("main: Graceful shutdown did not complete in %v: %v", timeout, err)
			err = api.Close()
		}

		if err != nil {
			log.Fatalf("main: could not stop server gracefully: %v", err)
		}
	}
}

// Echo is a basic HTTP Handler
func Echo(w http.ResponseWriter, r *http.Request) {
	n := rand.Intn(1000)
	log.Println("start", n)
	defer log.Println("end", n)

	time.Sleep(6 * time.Second)
	fmt.Fprintf(w, "You asked to %s %s \n", r.Method, r.URL.Path)
}
