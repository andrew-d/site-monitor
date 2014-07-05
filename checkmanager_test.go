package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var _ = fmt.Printf
var _ = suite.Run

type TestCheck struct {
	*Check
	UpdateCalls int
	UpdateError error
}

func (t *TestCheck) Update() error {
	t.UpdateCalls += 1
	return t.UpdateError
}

func TestKeyFor(t *testing.T) {
	assert.Equal(t, []byte{1, 0, 0, 0, 0, 0, 0, 0}, KeyFor(uint(1)))
	assert.Equal(t, []byte{1, 0, 0, 0, 0, 0, 0, 0}, KeyFor(uint64(1)))

	assert.Panics(t, func() {
		KeyFor("asdf")
	})

	assert.Panics(t, func() {
		KeyFor(uint16(1))
	})

	assert.Panics(t, func() {
		KeyFor(uint8(1))
	})
}
