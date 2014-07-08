// +build windows

package graceful

import (
	"os"
	"os/signal"
)

func registerShutdown(c chan os.Signal) {
	// TODO: do we want to catch both of these on Windows?
	signal.Notify(c, os.Interrupt, os.Kill)
}
