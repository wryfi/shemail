package config

import (
	"fmt"
	"runtime"
)

var ShemailVersion, GitRevision, BuildDate string

// PrintShemailVersion prints the version information of the current binary,
// provided such info was passed in via LDFLAGS. See justfile, for example.
func PrintShemailVersion() {
	fmt.Println("shemail")
	fmt.Println("-------")
	fmt.Println(" commit: ", GitRevision)
	fmt.Println("   date: ", BuildDate)
	fmt.Println(" golang: ", runtime.Version())
	fmt.Println("version: ", ShemailVersion)
}
