// +build linux

package main

import (
	"bastion/internal/proxy"
	"fmt"
	"os"
)

var Version string

func main() {
	fmt.Printf("Bastion proxy version %s starting...\n\n", Version)
	app, err := proxy.New()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	app.Run()
	app.Shutdown()
}
