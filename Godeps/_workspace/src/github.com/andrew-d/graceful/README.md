graceful [![GoDoc](https://godoc.org/github.com/andrew-d/graceful?status.png)](http://godoc.org/github.com/andrew-d/graceful) [![Build Status](https://travis-ci.org/andrew-d/graceful.svg)](https://travis-ci.org/andrew-d/graceful)
========

This is a fork of [Stretchr, Inc.'s graceful](https://github.com/stretchr/graceful),
a Go 1.3+ package enabling graceful shutdown of http.Handler servers.  This fork
allows more fine-grained control over when the server is shutdown.

## Usage

Usage of Graceful is simple. Create your http.Handler and pass it to the `Run` function:


```go

import (
	"fmt"
	"net/http"

	"github.com/andrew-d/graceful"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Welcome to the home page!")
	})

	// This will return when the server has shut down.
	graceful.Run(":3001", mux)
}
```

Or, create an instance of graceful.GracefulServer, configure the timeout, and
have more control over when your server shuts down:

```go

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andrew-d/graceful"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Welcome to the home page!")
	})

	srv := graceful.NewServer()
	srv.Timeout = 60 * time.Second

	go func() {
		// ... catch signals, or otherwise wait to signal shutdown
		srv.Shutdown <- struct{}{}
	}()

	// This will return when the server has shut down.
	srv.Run(":3001", mux)
}
```

In addition to `Run` there are the http.Server counterparts `ListenAndServe`,
`ListenAndServeTLS` and `Serve`, which allow additional configuration.  See
[the examples](https://github.com/andrew-d/graceful/tree/master/examples)
for some fully-working demonstrations.

## How It Works

When Graceful is asked to shutdown, it:

1. Disables Keep-Alive connections.
2. Closes the listening socket, allowing another process to listen on that port
   immediately.
3. Starts a timer of `timeout` duration to give active requests a chance to finish.
4. If the timeout expires, forcefully closes all active connections.
5. Returns from the function, allowing the server to terminate.

## Notes

- If the `timeout` value is 0, the server never times out, allowing all active
  requests to complete.
- Sending to the 'Shutdown' channel a second time will forcefully close all open
  connections.
