package main

import (
	"flag"
	"fmt"
	//"strconv"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/relab/smartMerge/regserver"
)

var (
	port  = flag.Int("port", 10000, "this servers address ip:port.")
	gcoff = flag.Bool("gcoff", false, "turn garbage collection off.")
	alg   = flag.String("alg", "", "algorithm to use (sm | dyna | cons )")
	allCores       = flag.Bool("all-cores", false, "use all available logical CPUs")
)

func main() {
	flag.Parse()

	if *gcoff {
		debug.SetGCPercent(-1)
	}

	if *allCores {
		cpus := runtime.NumCPU()
		runtime.GOMAXPROCS(cpus)
	}

	var err error
	fmt.Println("Starting Server with port: ", *port)
	switch *alg {
	case "", "sm":
		_, err = regserver.StartAdv(*port)
	case "dyna":
		_, err = regserver.StartDyna(*port)
	case "cons":
		_, err = regserver.StartCons(*port)
	}

	if err != nil {
		fmt.Println(err)
		panic("Starting server returned error")
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	for {
		select {
		case signal := <-signalChan:
			if exit := handleSignal(signal); exit {
				err = regserver.Stop()
				if err != nil {
					fmt.Printf("Stopping server returned error: %v\n", err)
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
