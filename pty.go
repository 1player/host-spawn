package main

// Most of the logic from https://github.com/creack/pty/blob/master/pty_linux.go

import (
	"os"

	"fmt"
	"io"
	"sync"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

type pty struct {
	wg sync.WaitGroup

	previousStdinTermios unix.Termios

	master *os.File
	slave  *os.File
}

// TODO: close fds on error
func createPty() (*pty, error) {
	master, slave, err := termios.Pty()
	if err != nil {
		return nil, err
	}

	return &pty{
		master: master,
		slave:  slave,
	}, nil
}

func (p *pty) Stdin() *os.File {
	return p.slave
}

func (p *pty) Stdout() *os.File {
	return p.slave
}

func (p *pty) Stderr() *os.File {
	return p.slave
}

func (p *pty) Start() error {
	err := p.makeStdinRaw()
	if err != nil {
		return err
	}

	p.wg.Add(2)

	go func() {
		n, err := io.Copy(p.master, os.Stdin)
		fmt.Println(n, err)
		p.wg.Done()
	}()

	go func() {
		n, err := io.Copy(os.Stdout, p.master)
		fmt.Println(n, err)
		p.wg.Done()
	}()

	return nil
}

func (p *pty) Terminate() {
	p.restoreStdin()

	//p.slave.Close()
	p.master.Close()

	// TODO: somehow I can't figure out how to have the
	// spawned process send an EOF when its fds are closed,
	// so for this reason the io.Copy calls above never return.
	// Let's just quit, but we might lose data.
	//p.wg.Wait()
}

func (p *pty) makeStdinRaw() error {
	var stdinTermios unix.Termios
	if err := termios.Tcgetattr(os.Stdin.Fd(), &stdinTermios); err != nil {
		return err
	}

	p.previousStdinTermios = stdinTermios
	termios.Cfmakeraw(&stdinTermios)
	if err := termios.Tcsetattr(os.Stdin.Fd(), termios.TCSANOW, &stdinTermios); err != nil {
		return err
	}

	return nil
}

func (p *pty) restoreStdin() {
	_ = termios.Tcsetattr(os.Stdin.Fd(), termios.TCSANOW, &p.previousStdinTermios)
}
