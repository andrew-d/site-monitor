package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/andrew-d/graceful"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Welcome to the home page!\n")
	})

	// Create a server and set a longer timeout.
	srv := graceful.NewServer()
	srv.Timeout = 60 * time.Second

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		// Wait for a signal, then shutdown
		<-c
		srv.Shutdown <- struct{}{}
	}()

	// This will return when the server has shut down.
	fmt.Println("Starting server on port 3001...")
	srv.Run(":3001", mux)
	fmt.Println("Finished")
}
