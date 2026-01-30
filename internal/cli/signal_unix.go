//go:build !windows

package cli

import (
	"os"
	"os/signal"
	"syscall"
)

func newWinchChan() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	return ch
}
