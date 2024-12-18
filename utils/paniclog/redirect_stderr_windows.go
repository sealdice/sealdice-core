// Copied from https://github.com/rclone/rclone/tree/master/fs/log
// Log the panic under windows to the log file
//
// Code from minix, via
//
// https://play.golang.org/p/kLtct7lSUg

//go:build windows

package paniclog

import (
	"os"
	"syscall"

	log "sealdice-core/utils/kratos"
)

var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procSetStdHandle = kernel32.MustFindProc("SetStdHandle")
)

func setStdHandle(stdhandle int32, handle syscall.Handle) error {
	r0, _, e1 := syscall.SyscallN(procSetStdHandle.Addr(), uintptr(stdhandle), uintptr(handle))
	if r0 == 0 {
		if e1 != 0 {
			return error(e1)
		}
		return syscall.EINVAL
	}
	return nil
}

// redirectStderr to the file passed in
func redirectStderr(f *os.File) {
	err := setStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(f.Fd()))
	if err != nil {
		log.Fatalf("Failed to redirect stderr to file: %v", err)
	}
	// https://stackoverflow.com/questions/34772012/capturing-panic-in-golang rclone can't get some
	// I did some more experimenting and on window's you must also do os.Stderr = f since SetStdHandle does not affect the prior reference to stderr.
	// On unix it is not necessary since the Dup2 does affect the prior reference to stderr. ( Tim Lewis Commented)
	os.Stderr = f
}
