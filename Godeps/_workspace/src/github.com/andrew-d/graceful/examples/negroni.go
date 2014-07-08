package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/andrew-d/graceful"
	"github.com/codegangsta/negroni"
)

func main() {
	var wg sync.WaitGroup

	srv1 := graceful.NewServer()
	srv2 := graceful.NewServer()
	srv3 := graceful.NewServer()

	wg.Add(3)
	go func() {
		n := negroni.New()
		fmt.Println("Launching server on :3000")
		srv1.Run(":3000", n)
		fmt.Println("Terminated server on :3000")
		wg.Done()
	}()

	go func() {
		n := negroni.New()
		fmt.Println("Launching server on :3001")
		srv2.Run(":3001", n)
		fmt.Println("Terminated server on :3001")
		wg.Done()
	}()

	go func() {
		n := negroni.New()
		fmt.Println("Launching server on :3002")
		srv3.Run(":3002", n)
		fmt.Println("Terminated server on :3002")
		wg.Done()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		for _ = range c {
			srv1.Shutdown <- struct{}{}
			srv2.Shutdown <- struct{}{}
			srv3.Shutdown <- struct{}{}

			signal.Stop(c)
			close(c)
		}
	}()

	fmt.Println("Press Ctrl+C. All servers should terminate.")
	wg.Wait()
	fmt.Println("Finished!")
}
