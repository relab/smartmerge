package main

import (
	"flag"

	"github.com/relab/smartMerge/util"
)

var (
	confFile = flag.String("conf","config", "the config file, a list of host:port addresses.")
	prnt = flag.Bool("print", true, "print to screen")
)

func main() {
	flag.Parse()
	util.GetProcs(*confFile, *prnt)
}