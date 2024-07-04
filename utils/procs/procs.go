package procs

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/fyrchik/go-shlex"
)

type OutHandler func(string, string) string

type Process struct {
	CmdString string
	StdIn     io.WriteCloser
	Cmd       *exec.Cmd
	Dir       string
	Env       []string
	// show stdout, return value will be written to stdin
	OutputHandler OutHandler
}

func NewProcess(command string) *Process {
	return &Process{CmdString: command}
}

func (p *Process) Start() error {
	cmdparts, err := shlex.Split(strings.TrimSpace(p.CmdString))
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if len(cmdparts) == 1 {
		cmd = exec.Command(cmdparts[0])
	} else {
		cmd = exec.Command(cmdparts[0], cmdparts[1:]...)
	}
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}
	if p.Env != nil {
		cmd.Env = p.Env
	}
	p.Cmd = cmd
	p.Setpgid()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		return err
	}

	p.StdIn = stdin

	setupScanner := func(r io.Reader) *bufio.Scanner {
		scanner := bufio.NewScanner(r)
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, io.EOF
			}
			if i := bytes.IndexByte(data, '\n'); i >= 0 {
				return i + 1, data[0 : i+1], nil
			}
			if atEOF {
				return len(data), data, nil
			}
			return len(data), data, err
			// return 0, nil, nil
		})
		return scanner
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic:", r)
			}
		}()

		scanner := setupScanner(stdout)

		for scanner.Scan() {
			line := scanner.Text()
			if p.OutputHandler != nil {
				back := p.OutputHandler(line, "stdout")
				if back != "" {
					_, _ = stdin.Write([]byte(back))
				}
			}
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic:", r)
			}
		}()

		scanner := setupScanner(stderr)

		for scanner.Scan() {
			line := scanner.Text()
			if p.OutputHandler != nil {
				back := p.OutputHandler(line, "stderr")
				if back != "" {
					_, _ = stdin.Write([]byte(back))
				}
			}
		}
	}()
	return nil
}

func (p *Process) Wait() error {
	return p.Cmd.Wait()
}

func (p *Process) Stop() error {
	cmd := p.Cmd
	if cmd.ProcessState != nil {
		return nil
	}

	err := cmd.Process.Kill()
	if err != nil {
		return err
	}

	return nil
}
