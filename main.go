package main

import (
	"log"
	"os"

	"github.com/godbus/dbus/v5"
)

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
	} else {
		return 0, false
	}
}

func runCommandSync(args []string) (int, bool) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	// Subscribe to HostCommandExited messages
	if err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.Flatpak.Development"),
		dbus.WithMatchMember("HostCommandExited"),
	); err != nil {
		log.Fatalln(err)
	}
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)

	// Spawn host command
	proxy := conn.Object("org.freedesktop.Flatpak", "/org/freedesktop/Flatpak/Development")

	var pid uint32
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	cwd_path := nullTerminatedByteString(cwd)

	argv := make([][]byte, len(args))
	for i, arg := range args {
		argv[i] = nullTerminatedByteString(arg)
	}
	envs := map[string]string{"TERM": os.Getenv("TERM")}

	pty, err := createPty()
	if err != nil {
		log.Fatalln(err)
	}
	pty.Start()
	defer pty.Terminate()

	fds := map[uint32]dbus.UnixFD{
		0: dbus.UnixFD(pty.Stdin().Fd()),
		1: dbus.UnixFD(pty.Stdout().Fd()),
		2: dbus.UnixFD(pty.Stderr().Fd()),
	}

	flags := uint32(0)

	err = proxy.Call("org.freedesktop.Flatpak.Development.HostCommand", 0,
		cwd_path, argv, fds, envs, flags,
	).Store(&pid)
	if err != nil {
		log.Fatalln(err)
	}

	// Wait for HostCommandExited to fire
	for message := range signals {
		waitStatus := message.Body[1].(uint32)
		return interpretWaitStatus(waitStatus)
	}

	panic("unreachable")
}

func main() {
	if exitCode, exited := runCommandSync(os.Args[1:]); exited {
		os.Exit(exitCode)
	} else {
		os.Exit(255)
	}
}
