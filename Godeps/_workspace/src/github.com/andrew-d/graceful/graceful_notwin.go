// +build !windows

package graceful

import (
	"os"
	"os/signal"
	"syscall"
)

func registerNotify(c chan os.Signal) {
	// SIGTERM is a standard method of asking a process to
	// terminate 'nicely'.
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
}
