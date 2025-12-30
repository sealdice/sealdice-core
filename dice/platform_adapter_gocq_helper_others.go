//go:build !windows

package dice

import "os"

type ProcessExitGroup uintptr

func NewProcessExitGroup() (ProcessExitGroup, error) {
	return 0, nil //nolint:nilnil
}

func (g ProcessExitGroup) Dispose() error {
	return nil
}

func (g ProcessExitGroup) AddProcess(_ *os.Process) error {
	return nil
}
