//go:build windows

package signals

import "syscall"

var signals = [...]string{
	//Omit line n here....
	/**Compatible with Windows start*/
	16: "SIGUSR1",
	17: "SIGUSR2",
	18: "SIGTSTP",
	/**Compatible with windows end*/
}

/**Compatible with Windows start*/
func Kill(...interface{}) {
	return
}

const (
	SIGUSR1 = syscall.Signal(0x10)
	SIGUSR2 = syscall.Signal(0x11)
	SIGTSTP = syscall.Signal(0x12)
)
