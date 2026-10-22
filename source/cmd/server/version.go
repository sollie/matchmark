package main

import (
	"fmt"
	"io"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func versionString() string {
	return fmt.Sprintf("matchmark %s (commit %s, built %s)", version, commit, date)
}

func isVersionRequest(args []string) bool {
	return len(args) == 1 && (args[0] == "--version" || args[0] == "version")
}

func printVersion(w io.Writer) {
	_, _ = fmt.Fprintln(w, versionString())
}
