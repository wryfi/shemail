package main

import "github.com/wryfi/shemail/cli"

func main() {
	cmd := cli.SheMailCommand()
	if err := cli.Execute(cmd); err != nil {
		panic(err)
	}
}
