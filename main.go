package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

// Version is the current value injected at build time.
var Version string = "HEAD"

// blocklist contains the list of programs not working well with an allocated pty.
var blocklist = map[string]bool{
	"gio":       true,
	"podman":    true,
	"kde-open":  true,
	"kde-open5": true,
	"xdg-open":  true,
}

// Command line options
var flagPty = flag.Bool("pty", false, "Force allocate a pseudo-terminal for the host process")
var flagNoPty = flag.Bool("no-pty", false, "Do not allocate a pseudo-terminal for the host process")
var flagVersion = flag.Bool("version", false, "Show this program's version")
var flagEnvironmentVariables = flag.String("env", "TERM", "Comma separated list of environment variables to pass to the host process.")
var flagWorkingDirectory = flag.String("cwd", "", "Change working directory of the spawned process")

const OUR_BASENAME = "host-spawn"

// The exit code we return to identify an error in host-spawn itself,
// rather than in the host process
const OUR_EXIT_CODE = 127

func parseArguments() {
	const USAGE_PREAMBLE = `Usage: %s [options] [ COMMAND [ arguments... ] ]

Run COMMAND on your host machine from inside a flatpak sandbox or container.
If COMMAND is not set, spawn the user's default shell on the host.

Accepted options:
`
	const USAGE_FOOTER = `--

A pseudo-terminal will be automatically allocated unless stdout is
redirected or the command is known for misbehaving when attached to a
pty. (See https://github.com/1player/host-spawn/issues/12)
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
	var args []string

	basename := path.Base(os.Args[0])

	// Check if we're shimming a host command
	if strings.HasPrefix(basename, OUR_BASENAME) {
		parseArguments()
		args = flag.Args()

		// If no command is given, spawn a shell
		if len(args) == 0 {
			args = []string{"sh", "-c", "$SHELL"}
		}
	} else {
		args = append([]string{basename}, os.Args[1:]...)
	}

	// Allocate a pty if:
	// - stdout isn't redirected
	// - this isn't a blocklisted program
	// Any of the --pty or --no-pty options will take precedence

	allocatePty := !isStdoutRedirected() && !blocklist[args[0]]

	if *flagPty {
		allocatePty = true
	} else if *flagNoPty {
		allocatePty = false
	}

	// Get working directory
	var wd string
	if *flagWorkingDirectory != "" {
		wd = *flagWorkingDirectory
	} else {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(OUR_EXIT_CODE)
		}
	}

	// Lookup and passthrough environment variables
	envVars := make(map[string]string)
	for _, k := range strings.Split(*flagEnvironmentVariables, ",") {
		if v, ok := os.LookupEnv(k); ok {
			envVars[k] = v
		}
	}

	// OK, let's go
	command := Command{
		Args:             args,
		WorkingDirectory: wd,
		AllocatePty:      allocatePty,
		EnvVars:          envVars,
	}

	exitCode, err := command.SpawnAndWait()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = OUR_EXIT_CODE
	}

	os.Exit(exitCode)
}
