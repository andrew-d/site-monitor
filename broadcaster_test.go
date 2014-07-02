package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var _ = fmt.Println

type TestBroadcaster struct {
	*Broadcaster
	closed bool
}

func (t *TestBroadcaster) process() {
	t.Broadcaster.process()
	t.closed = true
}

func NewTestBroadcaster() *TestBroadcaster {
	b := &TestBroadcaster{
		Broadcaster: NewBroadcaster(),
		closed:      false,
	}
	go b.process()
	return b
}

func TestBroadcasterClosed(t *testing.T) {
	t.Parallel()

	b := NewTestBroadcaster()
	b.Close()

	time.Sleep(10 * time.Millisecond)
	assert.True(t, b.closed)
}

// Asserts that broadcasting will work properly, including:
//	- Getting the right # of messages
//	- Each listener closing when the broadcaster does
//	- No messages lost
func TestBroadcasterBasic(t *testing.T) {
	t.Parallel()

	seen := map[int]int{}
	listeners := []*BroadcastListener{}
	closed := []chan struct{}{}
	mutex := &sync.Mutex{}

	b := NewBroadcaster()

	// Create 3 listeners and have them track all the seen values
	for i := 0; i < 3; i++ {
		l := b.Listen()

		go func() {
			cl := make(chan struct{})

			mutex.Lock()
			listeners = append(listeners, l)
			closed = append(closed, cl)
			mutex.Unlock()

			for val := range l.Chan() {
				mutex.Lock()
				seen[val.(int)] += 1
				mutex.Unlock()
			}

			cl <- struct{}{}
			l.Close()
		}()
	}

	// Send 10 numbers through.
	for i := 0; i < 10; i++ {
		b.Write(i)
	}
	b.Close()

	// Wait for all the worker routines to exit.
	timeout := time.After(5 * time.Second)
	for i, c := range closed {
		select {
		case <-c:
			// ok
		case <-timeout:
			t.Fatalf("worker routine %d didn't close in time", i)
		}
	}

	// Verify contents
	for i := 0; i < 10; i++ {
		count, found := seen[i]
		assert.True(t, found, "key %d was not found", i)
		assert.Equal(t, 3, count, "key %d was seen %d times (expected 3)", i, count)
	}
	assert.Equal(t, 10, len(seen))
}

// Asserts that we can close a listener properly
func TestBroadcasterClosing(t *testing.T) {
	t.Parallel()

	closed1 := make(chan struct{})
	closed2 := make(chan struct{})
	got1 := []int{}
	got2 := []int{}

	b := NewBroadcaster()
	l := b.Listen()
	go func(){
		val := <-l.Chan()
		got1 = append(got1, val.(int))
		l.Close()
		closed1 <- struct{}{}
	}()

	l2 := b.Listen()
	go func(){
		for val := range l2.Chan() {
			got2 = append(got1, val.(int))
		}
		closed2 <- struct{}{}
	}()

	timeout := time.After(5 * time.Second)

	// Send the first value, and wait for the corresponding exit.
	b.Write(123)
	select {
	case <-closed1:
		// Pass
	case <-timeout:
		t.Fatalf("first worker didn't finish in time")
	}

	assert.Equal(t, 1, len(got1))
	assert.Equal(t, 123, got1[0])

	// Send the second, verify that we got it only in 1
	b.Write(456)
	b.Close()
	select {
	case <-closed2:
		// Pass
	case <-timeout:
		t.Fatalf("second worker didn't finish in time")
	}

	assert.Equal(t, 1, len(got1))
	assert.Equal(t, 123, got1[0])

	assert.Equal(t, 2, len(got2))
	assert.Equal(t, 123, got2[0])
	assert.Equal(t, 456, got2[1])
}
