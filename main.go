package main

import (
	"os"

	"github.com/EkeMinusYou/gelf/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}