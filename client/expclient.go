package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/relab/goxos/kvs/bgen"
	lat "github.com/relab/smartMerge/directCombineLattice"
	"github.com/relab/smartMerge/dynaclient"
	"github.com/relab/smartMerge/elog"
	e "github.com/relab/smartMerge/elog/event"
	"github.com/relab/smartMerge/rpc"
	"github.com/relab/smartMerge/smclient"
	"github.com/relab/smartMerge/util"
)

var (
	//General
	gcOff    = flag.Bool("gc-off", false, "turn garbage collection off")
	showHelp = flag.Bool("help", false, "show this help message and exit")

	// Mode
	mode = flag.String("mode", "", "run mode: (user | bench )")

	//Config
	confFile = flag.String("conf", "config", "the config file, a list of host:port addresses.")
	clientid = flag.Int("id", 0, "the client id")
	nclients = flag.Int("nclients", 1, "the number of clients")
	initsize = flag.Int("initsize", 1, "the number of servers in the initial configuration")

	alg = flag.String("alg", "", "algorithm to be used: (sm | dyna | cons)")

	contW  = flag.Bool("contW", false, "continuously write")
	contR  = flag.Bool("contR", false, "continuously read")
	reads  = flag.Int("reads", 0, "number of reads to be performed.")
	writes = flag.Int("writes", 0, "number of writes to be performed.")
	size   = flag.Int("size", 16, "number of bytes for value.")

	doelog = flag.Bool("elog", false, "log latencies in user mode.")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	parseFlags()

	if *gcOff {
		debug.SetGCPercent(-1)
	}

	switch *mode {
	case "", "user":
		usermain()
	case "bench":
		expmain()
	default:
		fmt.Fprintf(os.Stderr, "Unkown mode specified: %q\n", *mode)
		flag.Usage()
	}
}

func expmain() {
	parseFlags()

	//Turn garbage collection off.
	if *gcOff {
		debug.SetGCPercent(-1)
	}

	//Parse Processes from Config file.
	addrs, ids := util.GetProcs(*confFile, true)

	//Build initial blueprint.
	if *initsize > len(ids) {
		fmt.Fprintln(os.Stderr, "Not enough servers to fulfill initsize.")
		return
	}

	iadd := make(map[lat.ID]bool, *initsize)

	for i := 0; i < *initsize; i++ {
		iadd[lat.ID(ids[i])] = true
	}

	initBlp := &lat.Blueprint{Add: iadd, Rem: nil}

	var wg sync.WaitGroup

	elog.Enable()
	defer elog.Flush()
	stop := make(chan struct{}, *nclients)

	for i := 0; i < *nclients; i++ {
		fmt.Println("starting client number: ", i)
		cl, mgr, err := NewClient(addrs, initBlp, *alg, (*clientid)+i)
		if err != nil {
			fmt.Println("Error creating client: ", err)
			continue
		}

		defer PrintErrors(mgr)
		wg.Add(1)
		switch {
		case *contW:
			go contWrite(cl, *size, stop, &wg)
		case *contR:
			go contRead(cl, stop, &wg)
		case *reads > 0:
			go doReads(cl, *reads, &wg)
		case *writes > 0:
			go doWrites(cl, *size, *writes, &wg)
		}
	}

	if *contR || *contW {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	loopSignals:
		for {
			select {
			case signal := <-signalChan:
				if exit := handleSignal(signal); exit {
					fmt.Println("stopping goroutines")
					close(stop)
					break loopSignals //break for loop, not only select
				}
			}
		}
	}
	fmt.Println("waiting for goroutines")
	wg.Wait()
	fmt.Println("finished waiting")
	return
}

func NewClient(addrs []string, initB *lat.Blueprint, alg string, id int) (cl RWRer, mgr *rpc.Manager, err error) {
	mgr, err = rpc.NewManager(addrs)
	if err != nil {
		fmt.Println("Creating manager returned error: ", err)
		return
	}
	switch alg {
	case "", "sm":
		cl, err = smclient.New(initB, mgr, uint32(id))
	case "dyna":
		cl, err = dynaclient.New(initB, mgr, uint32(id))
	case "cons":
		return nil, nil, errors.New("Consensus based algorithm not implemented yet.")
	}
	return
}

func contWrite(cl RWRer, size int, stop chan struct{}, wg *sync.WaitGroup) {
	fmt.Println("starting continous write")
	var (
		value   = make([]byte, size)
		cnt     int
		reqsent time.Time
	)

	bgen.GetBytes(value)
	cchan := make(chan int, 1)
loop:
	for {
		reqsent = time.Now()
		go func() {
			cchan <- cl.Write(value)
		}()
		select {
		case cnt = <-cchan:
			elog.Log(e.NewTimedEventWithMetric(e.ClientWriteLatency, reqsent, uint64(cnt)))
		case <-stop:
			break loop
		}
	}
	fmt.Println("finished continous write")
	wg.Done()
}

func contRead(cl RWRer, stop chan struct{}, wg *sync.WaitGroup) {
	fmt.Println("starting continous read")
	var (
		c       int
		cnt     int
		reqsent time.Time
	)

	cchan := make(chan int, 1)
loop:
	for {
		reqsent = time.Now()
		go func() {
			_, c = cl.Read()
			cchan <- c
		}()
		select {
		case <-stop:
			fmt.Println("received stopping signal")
			break loop
		case cnt = <-cchan:
			elog.Log(e.NewTimedEventWithMetric(e.ClientReadLatency, reqsent, uint64(cnt)))
		}
	}
	fmt.Println("finished continous read")
	wg.Done()
}

func doWrites(cl RWRer, size int, writes int, wg *sync.WaitGroup) {
	var (
		value   = make([]byte, size)
		cnt     int
		reqsent time.Time
	)

	bgen.GetBytes(value)
	for i := 0; i < writes; i++ {
		reqsent = time.Now()
		cnt = cl.Write(value)
		elog.Log(e.NewTimedEventWithMetric(e.ClientWriteLatency, reqsent, uint64(cnt)))
	}
	fmt.Println("finished writes")
	wg.Done()
}

func doReads(cl RWRer, reads int, wg *sync.WaitGroup) {
	var (
		cnt     int
		reqsent time.Time
	)

	for i := 0; i < reads; i++ {
		reqsent = time.Now()
		_, cnt = cl.Read()
		elog.Log(e.NewTimedEventWithMetric(e.ClientReadLatency, reqsent, uint64(cnt)))
	}
	fmt.Println("finished reads")
	wg.Done()
}

func parseFlags() {
	flag.Usage = Usage
	flag.Parse()
	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}
}

func handleSignal(signal os.Signal) bool {
	fmt.Println("received signal,", signal)
	switch signal {
	case os.Interrupt, os.Kill, syscall.SIGTERM:
		return true
	default:
		//glog.Warningln("unhandled signal", signal)
		return false
	}
}

type RWRer interface {
	Read() ([]byte, int)
	Write(val []byte) int
	Reconf(prop *lat.Blueprint) (int, error)
	GetCur() *lat.Blueprint
}
