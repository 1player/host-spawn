// Flatpak spawn simple reimplementation
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

const USAGE = `Usage: %s [options] COMMAND [arguments...]
		
Accepted options:
	--no-pty	do not allocate a pseudo-terminal for the host process
`

// Version is the current value injected at build time.
var Version string

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

func runCommandSync(args []string, allocatePty bool) (int, error) {

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
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)

	// Spawn host command
	proxy := conn.Object("org.freedesktop.Flatpak", "/org/freedesktop/Flatpak/Development")

	var pid uint32
	cwd, err := os.Getwd()
	if err != nil {
		return 0, err
	}

	cwdPath := nullTerminatedByteString(cwd)

	argv := make([][]byte, len(args))
	for i, arg := range args {
		argv[i] = nullTerminatedByteString(arg)
	}
	envs := map[string]string{"TERM": os.Getenv("TERM")}
	fds := map[uint32]dbus.UnixFD{
		0: dbus.UnixFD(os.Stdin.Fd()),
		1: dbus.UnixFD(os.Stdout.Fd()),
		2: dbus.UnixFD(os.Stderr.Fd()),
	}
	if allocatePty {
		pty, err := createPty()
		if err != nil {
			return 0, err
		}
		pty.Start()
		defer pty.Terminate()

		fds[0] = dbus.UnixFD(pty.Stdin().Fd())
		fds[1] = dbus.UnixFD(pty.Stdout().Fd())
		fds[2] = dbus.UnixFD(pty.Stderr().Fd())
	}

	flags := uint32(0)

	// Call command on the host
	err = proxy.Call("org.freedesktop.Flatpak.Development.HostCommand", 0,
		cwdPath, argv, fds, envs, flags,
	).Store(&pid)

	// an error occurred this early, most likely command not found.
	if err != nil {
		return 0, err
	}

	// Wait for HostCommandExited to fire
	for message := range signals {
		waitStatus := message.Body[1].(uint32)
		status, exited := interpretWaitStatus(waitStatus)
		if exited {
			return status, nil
		} else {
			return status, errors.New("child process did not terminate cleanly")
		}
	}

	panic("unreachable")
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, USAGE, os.Args[0])
		return
	}

	// Version flag
	if os.Args[1] == "-v" || os.Args[1] == "--version" {
		fmt.Println(Version)
		return
	}

	allocatePty := true
	command := os.Args[1:]

	if os.Args[1] == "--no-pty" {
		command = os.Args[2:]
		allocatePty = false
	}

	exitCode, err := runCommandSync(command, allocatePty)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 127
	}

	os.Exit(exitCode)
}
