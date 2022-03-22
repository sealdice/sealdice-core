//go:build !windows
// +build !windows

package dice

import "os"

type ProcessExitGroup uintptr

func NewProcessExitGroup() (ProcessExitGroup, error) {
	return 0, nil
}

func (g ProcessExitGroup) Dispose() error {
	return nil
}

func (g ProcessExitGroup) AddProcess(p *os.Process) error {
	return nil
}
