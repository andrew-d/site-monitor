package graceful

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"syscall"
	"testing"
	"time"
)

const KILL_TIME = 50 * time.Millisecond

func runQuery(t *testing.T, expected int, shouldErr bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	client := http.Client{}
	r, err := client.Get("http://localhost:3000")

	if shouldErr {
		if err == nil {
			t.Fatal("Expected an error but none was encountered")
		} else {
			if err.(*url.Error).Err == io.EOF {
				return
			}

			errno := err.(*url.Error).Err.(*net.OpError).Err.(syscall.Errno)
			if errno == syscall.ECONNREFUSED {
				return
			} else if err != nil {
				t.Fatal("Error on GET:", err)
			}
		}
	}

	if r != nil && r.StatusCode != expected {
		t.Fatalf("Incorrect status code on response: (expected) %d != %d (actual)",
			expected, r.StatusCode)
	} else if r == nil {
		t.Fatal("No response when a response was expected")
	}
}

func runServer(t *testing.T, timeout, sleep time.Duration) (srv *GracefulServer, wg *sync.WaitGroup) {
	wg = &sync.WaitGroup{}
	srv = NewServer()

	srv.Timeout = timeout

	wg.Add(1)
	go func() {
		defer wg.Done()

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
			time.Sleep(sleep)
			rw.WriteHeader(http.StatusOK)
		})

		if err := srv.Run(":3000", mux); err != nil {
			t.Fatal("Error from graceful run: %s", err)
		}
	}()
	return
}

func TestGracefulRun(t *testing.T) {
	srv, wg := runServer(t, KILL_TIME, KILL_TIME/2)

	for i := 0; i < 10; i++ {
		go runQuery(t, http.StatusOK, false, wg)
	}

	time.Sleep(10 * time.Millisecond)
	srv.Shutdown <- struct{}{}
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 10; i++ {
		go runQuery(t, 0, true, wg)
	}

	wg.Wait()
}

func TestGracefulRunTimesOut(t *testing.T) {
	srv, wg := runServer(t, KILL_TIME, KILL_TIME*10)

	for i := 0; i < 10; i++ {
		go runQuery(t, 0, true, wg)
	}

	time.Sleep(10 * time.Millisecond)
	srv.Shutdown <- struct{}{}
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 10; i++ {
		go runQuery(t, 0, true, wg)
	}

	wg.Wait()
}

func TestGracefulRunDoesntTimeOut(t *testing.T) {
	srv, wg := runServer(t, 0, KILL_TIME*2)

	for i := 0; i < 10; i++ {
		go runQuery(t, http.StatusOK, false, wg)
	}

	time.Sleep(10 * time.Millisecond)
	srv.Shutdown <- struct{}{}
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 10; i++ {
		go runQuery(t, 0, true, wg)
	}

	wg.Wait()
}

func TestGracefulRunNoRequests(t *testing.T) {
	srv, wg := runServer(t, 0, KILL_TIME*2)

	time.Sleep(10 * time.Millisecond)
	srv.Shutdown <- struct{}{}

	wg.Wait()
}
