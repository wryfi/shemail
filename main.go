package main

import (
	"github.com/wryfi/shemail/cli"
	"github.com/wryfi/shemail/logging"
)

var log = &logging.Logger

func main() {
	cmd := cli.SheMailCommand()
	if err := cli.Execute(cmd); err != nil {
		log.Fatal().Msgf("failed to run command: %v", err)
	}
}
