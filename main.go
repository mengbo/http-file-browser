package main

import (
	"os"

	"github.com/mengbo/http-file-browser/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
