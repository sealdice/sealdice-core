//go:build android
// +build android

package procs

import (
	"syscall"
)

func (p *Process) KillProcess() error {
	cmd := p.Cmd
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)

}
func (p *Process) Setpgid() {
	cmd := p.Cmd
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
