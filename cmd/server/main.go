// Command server runs the agent-protocol MCP proxy.
package main

import (
	"log"

	_ "go.uber.org/automaxprocs" // set GOMAXPROCS from the cgroup CPU quota

	"agent-protocol/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
