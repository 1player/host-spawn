// Create a pty for us
package main

import (
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"unsafe"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

type winsize struct {
	Rows uint16 // ws_row: Number of rows (in cells)
	Cols uint16 // ws_col: Number of columns (in cells)
	X    uint16 // ws_xpixel: Width in pixels
	Y    uint16 // ws_ypixel: Height in pixels
}

type pty struct {
	wg      sync.WaitGroup
	signals chan os.Signal

	previousStdinTermios unix.Termios

	master *os.File
	slave  *os.File
}

func createPty() (*pty, error) {
	master, slave, err := termios.Pty()
	if err != nil {
		return nil, err
	}

	return &pty{
		master:  master,
		slave:   slave,
		signals: make(chan os.Signal, 1),
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

	p.handleSignals()

	p.wg.Add(2)

	go func() {
		io.Copy(p.master, os.Stdin)
		p.wg.Done()
	}()

	go func() {
		io.Copy(os.Stdout, p.master)
		p.wg.Done()
	}()

	p.inheritWindowSize()

	return nil
}

func (p *pty) Terminate() {
	p.restoreStdin()

	p.master.Close()
	p.slave.Close()
	close(p.signals)

	// TODO: somehow I can't figure out how to have the
	// spawned process send an EOF when its fds are closed,
	// so for this reason the io.Copy calls above never return.
	//p.wg.Wait()
}

func (p *pty) handleSignals() {
	signal.Notify(p.signals, syscall.SIGWINCH)

	go func() {
		for signal := range p.signals {
			switch signal {
			case syscall.SIGWINCH:
				_ = p.inheritWindowSize()
			}
		}
	}()
}

func (p *pty) inheritWindowSize() error {
	var winsz winsize
	if err := getWinsz(os.Stdout, &winsz); err != nil {
		return err
	}
	if err := setWinsz(p.master, &winsz); err != nil {
		return err
	}
	return nil
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

func getWinsz(file *os.File, winsz *winsize) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(winsz)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func setWinsz(file *os.File, winsz *winsize) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(winsz)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}
