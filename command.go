package main

import (
	"errors"
	"os"

	"github.com/godbus/dbus/v5"
	"golang.org/x/sys/unix"
	"os/signal"
	"syscall"
)

type Command struct {
	Args             []string
	WorkingDirectory string
	AllocatePty      bool
	EnvVars          map[string]string

	proxy dbus.BusObject
	pty   *pty
	pid   uint32
}

func nullTerminatedByteString(s string) []byte {
	return append([]byte(s), 0)
}

// Extract exit code from waitpid(2) status
func interpretWaitStatus(status uint32) (int, bool) {
	// From /usr/include/bits/waitstatus.h
	WTERMSIG := status & 0x7f
	WIFEXITED := WTERMSIG == 0

	if WIFEXITED {
		WEXITSTATUS := (status & 0xff00) >> 8
		return int(WEXITSTATUS), true
	}

	return 0, false
}

func (c *Command) signal(signal syscall.Signal) error {
	return c.proxy.Call("org.freedesktop.Flatpak.Development.HostCommandSignal", 0,
		c.pid, uint32(signal), false,
	).Store()
}

func (c *Command) SpawnAndWait() (int, error) {
	// Connect to the dbus session to talk with flatpak-session-helper process.
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	// Subscribe to HostCommandExited messages
	if err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.Flatpak.Development"),
		dbus.WithMatchMember("HostCommandExited"),
	); err != nil {
		return 0, err
	}
	dbusSignals := make(chan *dbus.Signal, 1)
	conn.Signal(dbusSignals)

	// Spawn host command
	c.proxy = conn.Object("org.freedesktop.Flatpak", "/org/freedesktop/Flatpak/Development")

	cwdPath := nullTerminatedByteString(c.WorkingDirectory)

	argv := make([][]byte, len(c.Args))
	for i, arg := range c.Args {
		argv[i] = nullTerminatedByteString(arg)
	}

	fds := map[uint32]dbus.UnixFD{
		0: dbus.UnixFD(os.Stdin.Fd()),
		1: dbus.UnixFD(os.Stdout.Fd()),
		2: dbus.UnixFD(os.Stderr.Fd()),
	}
	if c.AllocatePty {
		c.pty, err = createPty()
		if err != nil {
			return 0, err
		}
		err = c.pty.Start()
		if err != nil {
			return 0, err
		}
		defer c.pty.Terminate()

		fds[0] = dbus.UnixFD(c.pty.Stdin().Fd())
		fds[1] = dbus.UnixFD(c.pty.Stdout().Fd())
		fds[2] = dbus.UnixFD(c.pty.Stderr().Fd())
	}

	flags := uint32(0)

	// Call command on the host
	err = c.proxy.Call("org.freedesktop.Flatpak.Development.HostCommand", 0,
		cwdPath, argv, fds, c.EnvVars, flags,
	).Store(&c.pid)

	// an error occurred this early, most likely command not found.
	if err != nil {
		return 0, err
	}

	return c.waitForSignals(dbusSignals)
}

// Wait for either host or DBus signals
func (c *Command) waitForSignals(dbusSignals chan *dbus.Signal) (int, error) {
	hostSignals := make(chan os.Signal, 1)
	signal.Notify(hostSignals)

	for {
		select {
		case signal := <-hostSignals:
			unixSignal := signal.(syscall.Signal)

			if unixSignal == unix.SIGWINCH && c.pty != nil {
				c.pty.inheritWindowSize()
				break
			} else if unixSignal == unix.SIGURG {
				// Ignore runtime-generated SIGURG messages
				// See https://github.com/golang/go/issues/37942
				break
			}

			// Send the signal but ignore any error, as there is
			// nothing much we can do about them
			_ = c.signal(unixSignal)

		case message := <-dbusSignals:
			// HostCommandExited has fired
			waitStatus := message.Body[1].(uint32)
			status, exited := interpretWaitStatus(waitStatus)
			if exited {
				return status, nil
			} else {
				return status, errors.New("child process did not terminate cleanly")
			}
		}
	}

	// unreachable
}
