package main

import (
	"flag"
	"fmt"
	//"strconv"
	"os"
	"os/signal"
	"syscall"

	"github.com/relab/smartMerge/regserver"
)

var (
	port = flag.Int("port", 10000, "this servers address ip:port.")
)

func main() {
	flag.Parse()
	fmt.Println("Starting Server with port: ", *port)
	_, err := regserver.StartAdv(*port)
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
