//go:build !android
// +build !android

package procs

func (p *Process) KillProcess() error {
	cmd := p.Cmd
	return cmd.Process.Kill()
}
func (p *Process) Setpgid() {
}
