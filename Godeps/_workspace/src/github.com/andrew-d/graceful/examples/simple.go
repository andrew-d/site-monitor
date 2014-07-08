package main

import (
	"fmt"
	"net/http"

	"github.com/andrew-d/graceful"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Welcome to the home page!\n")
	})

	// This will return when the server has shut down.
	// The default timeout is 10 seconds.
	fmt.Println("Starting server on port 3001...")
	graceful.Run(":3001", mux)
	fmt.Println("Finished")
}
