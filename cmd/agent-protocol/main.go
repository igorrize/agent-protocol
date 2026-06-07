// Command agent-protocol is the MCP proxy. Subcommands:
//
//	serve        run the proxy server (HTTP MCP + health)
//	orchestrate  launch a locked, interactive Claude orchestrator
package main

import (
	"fmt"
	"os"

	"agent-protocol/internal/app"
	"agent-protocol/internal/app/orchestrate"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "serve":
		if err := app.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "serve:", err)
			os.Exit(1)
		}
	case "orchestrate":
		if err := orchestrate.Run(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "orchestrate:", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: agent-protocol serve|orchestrate")
}
