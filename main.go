package main

import (
	"os"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core"
)

//go:generate make modgraph
func main() {
	os.Exit(core.Main())
}
