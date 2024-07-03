package main

import (
	"os"

	"github.com/eliran89c/klama/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
