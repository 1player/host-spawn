package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/godbus/dbus/v5"
)

// Version is the current value injected at build time.
var Version string = "HEAD"

// blocklist contains the list of programs not working well with an allocated pty.
var blocklist = map[string]bool{
	"xdg-open": true,
	"gio":      true,
}

// Command line options
var flagPty = flag.Bool("pty", false, "Force allocate a pseudo-terminal for the host process")
var flagNoPty = flag.Bool("no-pty", false, "Do not allocate a pseudo-terminal for the host process")
var flagVersion = flag.Bool("version", false, "Show this program's version")
var flagEnvironmentVariables = flag.String("env", "TERM", "Comma separated list of environment variables to pass to the host process.")

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

func runCommandSync(args []string, allocatePty bool, envsToPassthrough []string) (int, error) {

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

	envs := make(map[string]string)
	for _, e := range envsToPassthrough {
		if v, ok := os.LookupEnv(e); ok {
			envs[e] = v
		}
	}

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
		err = pty.Start()
		if err != nil {
			return 0, err
		}
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
	const USAGE_PREAMBLE = `Usage: %s [options] [ COMMAND [ arguments... ] ]

If COMMAND is not set, spawn a shell on the host.

Accepted options:
`
	const USAGE_FOOTER = `--

If neither pty option is passed, default to allocating a pseudo-terminal unless
the command is known for misbehaving when attached to a pty.

For more details visit https://github.com/1player/host-spawn/issues/12
`

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, USAGE_PREAMBLE, os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, USAGE_FOOTER)
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

		// If no command is given, spawn a shell
		if len(command) == 0 {
			command = []string{"sh", "-c", "$SHELL"}
		}
	} else {
		command = append([]string{basename}, os.Args[1:]...)
	}

	// Lookup if this is a blocklisted program, where we won't enable pty.
	allocatePty := !blocklist[command[0]]
	if *flagPty {
		allocatePty = true
	} else if *flagNoPty {
		allocatePty = false
	}

	envsToPassthrough := strings.Split(*flagEnvironmentVariables, ",")

	exitCode, err := runCommandSync(command, allocatePty, envsToPassthrough)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 127
	}

	os.Exit(exitCode)
}
