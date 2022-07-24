package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/godbus/dbus/v5"
)

// Version is the current value injected at build time.
var Version string = "HEAD"

// Command line options
var flagNoPty = flag.Bool("no-pty", false, "do not allocate a pseudo-terminal for the host process")
var flagVersion = flag.Bool("version", false, "show this program's version")

const OUR_BASENAME = "host-spawn"

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

func parseArguments() {
	const USAGE_PREAMBLE = `Usage: %s [options] COMMAND [arguments...]

Accepted options:
`

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, USAGE_PREAMBLE, os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	flag.Parse()

	if *flagVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
}

func main() {
	var command []string

	basename := path.Base(os.Args[0])

	// Check if we're shimming a host command
	if basename == OUR_BASENAME {
		parseArguments()
		command = flag.Args()
	} else {
		command = append([]string{basename}, os.Args[1:]...)
	}

	allocatePty := !*flagNoPty
	exitCode, err := runCommandSync(command, allocatePty)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 127
	}

	os.Exit(exitCode)
}
