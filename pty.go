// Create a pty for us
package main

import (
	"io"
	"os"
	"os/signal"
	"sync"

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
	stdinIsatty          bool

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
	signal.Notify(p.signals, unix.SIGWINCH)

	go func() {
		for signal := range p.signals {
			switch signal {
			case unix.SIGWINCH:
				_ = p.inheritWindowSize()
			}
		}
	}()
}

func (p *pty) inheritWindowSize() error {
	winsz, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return err
	}
	if err := unix.IoctlSetWinsize(int(p.master.Fd()), unix.TIOCSWINSZ, winsz); err != nil {
		return err
	}
	return nil
}

func (p *pty) makeStdinRaw() error {
	var stdinTermios unix.Termios

	err := termios.Tcgetattr(os.Stdin.Fd(), &stdinTermios)

	// We might get ENOTTY if stdin is redirected
	if err != nil {
		if errno, ok := err.(unix.Errno); ok {
			if errno == unix.ENOTTY {
				return nil
			} else {
				return err
			}
		}
	}

	p.previousStdinTermios = stdinTermios
	p.stdinIsatty = true

	termios.Cfmakeraw(&stdinTermios)
	if err := termios.Tcsetattr(os.Stdin.Fd(), termios.TCSANOW, &stdinTermios); err != nil {
		return err
	}

	return nil
}

func (p *pty) restoreStdin() {
	if p.stdinIsatty {
		_ = termios.Tcsetattr(os.Stdin.Fd(), termios.TCSANOW, &p.previousStdinTermios)
	}
}
