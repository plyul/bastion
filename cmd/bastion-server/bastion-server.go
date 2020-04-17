package main

import (
	"bastion/internal/server"
	"fmt"
)

var Version string

func main() {
	fmt.Printf("Bastion server version %s starting...\n\n", Version)
	app := server.New()
	app.Run()
	app.Shutdown()
}
