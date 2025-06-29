//go:build !windows

// fork from https://github.com/g-utils/endless

package signals

import "syscall"

const (
	SIGUSR1 = syscall.SIGUSR1
	SIGUSR2 = syscall.SIGUSR2
	SIGTSTP = syscall.SIGTSTP
)

var Kill func(pid int, sig syscall.Signal) error = syscall.Kill
