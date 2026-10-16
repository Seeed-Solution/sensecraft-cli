package main

import (
	"os"
	"sensecraft-cloud-terminal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
