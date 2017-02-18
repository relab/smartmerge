package main

import (
	"flag"
	//"strconv"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"syscall"

	"github.com/golang/glog"

	"github.com/relab/smartMerge/regserver"
)

var (
	port       = flag.Int("port", 10000, "this servers address ip:port.")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	gcoff      = flag.Bool("gcoff", false, "turn garbage collection off.")
	alg        = flag.String("alg", "", "algorithm to use (sm | dyna | ssr | cons )")
	allCores   = flag.Bool("all-cores", false, "use all available logical CPUs")

	abort = flag.Bool("abort", false, "abort rpcs on outdated configurations.")
)

func main() {
	flag.Parse()
	defer glog.Flush()

	if *gcoff {
		debug.SetGCPercent(-1)
	}

	if *cpuprofile != "" {
		glog.Infoln("Starting cpuprofiling in file", *cpuprofile)
		f, err := os.Create(*cpuprofile)
		if err != nil {
			glog.Errorln("err")
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *allCores {
		cpus := runtime.NumCPU()
		runtime.GOMAXPROCS(cpus)
	}

	var err error
	glog.Infoln("Starting Server with port: ", *port)
	switch *alg {
	case "", "sm":
		_, err = regserver.Start(*port, !(*abort))
	case "dyna":
		//_, err = regserver.StartDyna(*port)
	case "ssr":
		//_, err = regserver.StartSSR(*port)
	case "cons":
		_, err = regserver.Start(*port, !(*abort))
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
