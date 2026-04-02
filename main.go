package main

import "github.com/MaximusBenjamin/terminal-pomodoro/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
