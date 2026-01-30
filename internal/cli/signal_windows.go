//go:build windows

package cli

import "os"

func newWinchChan() <-chan os.Signal {
	return make(chan os.Signal)
}
