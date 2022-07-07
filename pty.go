package main

// Most of the logic from https://github.com/creack/pty/blob/master/pty_linux.go

import (
	"C"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)
import (
	"fmt"
	"io"
	"sync"

	"golang.org/x/term"
)

type pty struct {
	wg                sync.WaitGroup
	ptmx              *os.File
	oldStdinTermState *term.State
	Stdin             *os.File
	Stdout            *os.File
	Stderr            *os.File
}

// TODO: close fds on error
func createPty() (*pty, error) {
	ptmx, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	sname, err := ptsname(ptmx)
	if err != nil {
		return nil, err
	}

	if err := unlockpt(ptmx); err != nil {
		return nil, err
	}

	var p pty
	p.ptmx = ptmx

	if p.Stdin, err = os.OpenFile(sname, os.O_RDONLY, 0); err != nil {
		return nil, err
	}
	if p.Stdout, err = os.OpenFile(sname, os.O_WRONLY, 0); err != nil {
		return nil, err
	}
	if p.Stderr, err = os.OpenFile(sname, os.O_WRONLY, 0); err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *pty) Start() {
	var err error
	p.oldStdinTermState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	p.wg.Add(2)

	go func() {
		n, err := io.Copy(p.ptmx, os.Stdin)
		fmt.Println(n, err)
		p.wg.Done()
	}()

	go func() {
		n, err := io.Copy(os.Stdout, p.ptmx)
		fmt.Println(n, err)
		p.wg.Done()
	}()
}

func (p *pty) Terminate() {
	// Restore stdio state
	_ = term.Restore(int(os.Stdin.Fd()), p.oldStdinTermState)

	p.ptmx.Close()

	// TODO: somehow I can't figure out how to have the
	// spawned process send an EOF when its fds are closed,
	// so for this reason the io.Copy calls above never return.
	// Let's just quit, but we might lose data.
	//p.wg.Wait()
}

func ptsname(f *os.File) (string, error) {
	var n C.int
	err := ioctl(f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if err != nil {
		return "", err
	}
	return "/dev/pts/" + strconv.Itoa(int(n)), nil
}

func unlockpt(f *os.File) error {
	var u C.int
	// use TIOCSPTLCK with a pointer to zero to clear the lock
	return ioctl(f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&u)))
}

func ioctl(fd, cmd, ptr uintptr) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if e != 0 {
		return e
	}
	return nil
}
