package main

import (
	"os"

	"github.com/satocchi0416sh/dotgo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
