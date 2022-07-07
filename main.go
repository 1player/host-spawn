package main

import (
	"fmt"
	"log"
	"os"

	"github.com/godbus/dbus/v5"
)

func nullTerminatedByteString(s string) []byte {
	return append([]byte(s), 0)
}

func argvFromArguments() [][]byte {
	argv := make([][]byte, len(os.Args)-1)
	for i, arg := range os.Args[1:] {
		argv[i] = nullTerminatedByteString(arg)
	}

	return argv
}

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	proxy := conn.Object("org.freedesktop.Flatpak", "/org/freedesktop/Flatpak/Development")

	if err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.Flatpak.Development"),
		dbus.WithMatchMember("HostCommandExited"),
	); err != nil {
		log.Fatalln(err)
	}
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)

	var pid uint32
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	cwd_path := nullTerminatedByteString(cwd)
	argv := argvFromArguments()
	envs := map[string]string{"TERM": "xterm-256color"}
	fds := map[uint32]dbus.UnixFD{0: 0, 1: 1, 2: 2}
	flags := uint32(0)

	err = proxy.Call("org.freedesktop.Flatpak.Development.HostCommand", 0,
		cwd_path, argv, fds, envs, flags,
	).Store(&pid)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		select {
		case message := <-signals:
			exitStatus := message.Body[1].(uint32)
			fmt.Println(exitStatus)
			os.Exit(0)
		}
	}
}
