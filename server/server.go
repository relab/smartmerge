package main

import (
	"flag"
	//"strconv"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/golang/glog"

	"github.com/relab/smartMerge/regserver"
)

var (
	port     = flag.Int("port", 10000, "this servers address ip:port.")
	gcoff    = flag.Bool("gcoff", false, "turn garbage collection off.")
	alg      = flag.String("alg", "", "algorithm to use (sm | dyna | cons )")
	allCores = flag.Bool("all-cores", false, "use all available logical CPUs")

	noabort    = flag.Bool("no-abort", false, "do not send aborting new-cur information.")
)

func main() {
	flag.Parse()
	defer glog.Flush()

	if *gcoff {
		debug.SetGCPercent(-1)
	}

	if *allCores {
		cpus := runtime.NumCPU()
		runtime.GOMAXPROCS(cpus)
	}

	var err error
	glog.Infoln("Starting Server with port: ", *port)
	switch *alg {
	case "", "sm":
		_, err = regserver.StartAdv(*port, *noabort)
	case "dyna":
		_, err = regserver.StartDyna(*port)
	case "cons":
		_, err = regserver.StartCons(*port, *noabort)
	}

	if err != nil {
		glog.Fatalln("Starting server returned error", err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	for {
		select {
		case signal := <-signalChan:
			if exit := handleSignal(signal); exit {
				err = regserver.Stop()
				if err != nil {
					glog.Errorf("Stopping server returned error: %v\n", err)
				}
				return
			}
		}
	}
}

func handleSignal(signal os.Signal) bool {
	//log("received signal,", signal)
	switch signal {
	case os.Interrupt, os.Kill, syscall.SIGTERM:
		return true
	default:
		//glog.Warningln("unhandled signal", signal)
		return false
	}
}
