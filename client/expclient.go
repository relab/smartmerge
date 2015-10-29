package main

import (
	"runtime/pprof"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
	_ "net/http/pprof"
	"net/http"
	"log"

	"github.com/golang/glog"
	"github.com/relab/goxos/kvs/bgen"
	"github.com/relab/smartMerge/dynaclient"
	"github.com/relab/smartMerge/elog"
	e "github.com/relab/smartMerge/elog/event"
	"github.com/relab/smartMerge/smclient"
	"github.com/relab/smartMerge/consclient"
	"github.com/relab/smartMerge/util"
	qf "github.com/relab/smartMerge/qfuncs"
	pb "github.com/relab/smartMerge/proto"
	grpc "google.golang.org/grpc"
)

var (
	//General
	gcOff    = flag.Bool("gc-off", false, "turn garbage collection off")
	showHelp = flag.Bool("help", false, "show this help message and exit")
	allCores       = flag.Bool("all-cores", false, "use all available logical CPUs")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to this file")
	httpprof = flag.Bool("httpprof", false, "enable profiling via http server")

	// Mode
	mode = flag.String("mode", "", "run mode: (user | bench | exp )")
	alg = flag.String("alg", "", "algorithm to be used: (sm | dyna | odyna | cons )")
	doelog = flag.Bool("elog", false, "log latencies in user or exp mode.")

	//Config
	confFile = flag.String("conf", "config", "the config file, a list of host:port addresses.")
	clientid = flag.Int("id", 0, "the client id")
	nclients = flag.Int("nclients", 1, "the number of clients")
	initsize = flag.Int("initsize", 1, "the number of servers in the initial configuration")


	//Read or Write Bench
	contW  = flag.Bool("contW", false, "continuously write")
	contR  = flag.Bool("contR", false, "continuously read")
	reads  = flag.Int("reads", 0, "number of reads to be performed.")
	writes = flag.Int("writes", 0, "number of writes to be performed.")
	size   = flag.Int("size", 16, "number of bytes for value.")

	//Reconf Exp
	rm = flag.Bool("rm",false , "remove nclients servers concurrently.")
	add = flag.Bool("add", false, "add nclients servers concurrently")

)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	parseFlags()
	defer glog.Flush()	

	if *gcOff {
		glog.Infoln("Setting garbage collection to -1")
		debug.SetGCPercent(-1)
	}

	if *allCores {
		cpus := runtime.NumCPU()
		runtime.GOMAXPROCS(cpus)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Println("err")
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *httpprof {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}
	switch *mode {
	case "", "user":
		usermain()
	case "bench":
		benchmain()
	case "exp":
		expmain()
	default:
		fmt.Fprintf(os.Stderr, "Unkown mode specified: %q\n", *mode)
		flag.Usage()
	}
}

func benchmain() {
	parseFlags()

	//Turn garbage collection off.
	if *gcOff {
		debug.SetGCPercent(-1)
	}

	//Parse Processes from Config file.
	addrs, ids := util.GetProcs(*confFile, false)

	//Build initial blueprint.
	if *initsize > len(ids) && *initsize < 100 {
		glog.Errorln(os.Stderr, "Not enough servers to fulfill initsize.")
		return
	}

	initBlp := new(pb.Blueprint)
	if *initsize >= 100 {
		initBlp.Add = ids
	} else {
		initBlp.Add = ids[:*initsize]
	}
	

	var wg sync.WaitGroup

	elog.Enable()
	defer elog.Flush()
	stop := make(chan struct{}, *nclients)

	for i := 0; i < *nclients; i++ {
		glog.Infof("starting client number:  %d at time %v\n", i, time.Now())
		cl, mgr, err := NewClient(addrs, initBlp, *alg, (*clientid)+i)
		if err != nil {
			glog.Errorln("Error creating client: ", err)
			continue
		}

		defer LogErrors(mgr)
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
					glog.Infoln("stopping goroutines")
					close(stop)
					break loopSignals //break for loop, not only select
				}
			}
		}
	}
	glog.Infoln("waiting for goroutines")
	wg.Wait()
	glog.Infoln("finished waiting")
	return
}

func NewClient(addrs []string, initB *pb.Blueprint, alg string, id int) (cl RWRer, mgr *pb.Manager, err error) {
	mgr, err = pb.NewManager(addrs, pb.WithGrpcDialOptions(
		grpc.WithBlock(),
		grpc.WithTimeout(1000*time.Millisecond),
		grpc.WithInsecure()),
		pb.WithAReadSQuorumFunc(qf.AReadSQF),
		pb.WithAWriteSQuorumFunc(qf.AWriteSQF),
		pb.WithAWriteNQuorumFunc(qf.AWriteNQF),
		pb.WithSetCurQuorumFunc(qf.SetCurQF),
		pb.WithLAPropQuorumFunc(qf.LAPropQF),
		pb.WithSetStateQuorumFunc(qf.SetStateQF),
		pb.WithDReadSQuorumFunc(qf.DReadSQF),
		pb.WithDWriteSQuorumFunc(qf.DWriteSQF),
		pb.WithDWriteNSetQuorumFunc(qf.DWriteNSetQF),
		pb.WithDSetCurQuorumFunc(qf.DSetCurQF),
		pb.WithGetOneNQuorumFunc(qf.GetOneNQF),
		pb.WithCReadSQuorumFunc(qf.CReadSQF),
		pb.WithCWriteSQuorumFunc(qf.CWriteSQF),
		pb.WithCPrepareQuorumFunc(qf.CPrepareQF),
		pb.WithCAcceptQuorumFunc(qf.CAcceptQF),
		pb.WithCSetStateQuorumFunc(qf.CSetStateQF),
		pb.WithCWriteNQuorumFunc(qf.CWriteNQF),
	)
	if err != nil {
		glog.Errorln("Creating manager returned error: ", err)
		return
	}
	switch alg {
	case "", "sm":
		
		cl, err = smclient.New(initB, mgr, uint32(id))
	case "dyna":
		cl, err = dynaclient.New(initB, mgr, uint32(id))
	case "odyna": 
		cl, err = dynaclient.NewOrg(initB, mgr, uint32(id))
	case "cons":
		cl, err = consclient.New(initB, mgr, uint32(id))
	}
	return
}

func contWrite(cl RWRer, size int, stop chan struct{}, wg *sync.WaitGroup) {
	glog.Infoln("starting continous write")
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
			if cnt > 100 {
				break
			}
		case <-stop:
			break loop
		}
	}
	glog.Infoln("finished continous write")
	wg.Done()
}

func contRead(cl RWRer, stop chan struct{}, wg *sync.WaitGroup) {
	glog.Infoln("starting continous read")
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
			glog.Infoln("received stopping signal")
			break loop
		case cnt = <-cchan:
			elog.Log(e.NewTimedEventWithMetric(e.ClientReadLatency, reqsent, uint64(cnt)))
		}
	}
	glog.Infoln("finished continous read")
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
	glog.Infoln("finished writes")
	if wg != nil {
		wg.Done()
	}
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
	glog.Infoln("finished reads")
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
	glog.Infoln("received signal,", signal)
	switch signal {
	case os.Interrupt, os.Kill, syscall.SIGTERM:
		return true
	default:
		//glog.Warningln("unhandled signal", signal)
		return false
	}
}

type RWRer interface {
	RRead() ([]byte,int)
	Read() ([]byte, int)
	Write(val []byte) int
	Reconf(prop *pb.Blueprint) (int, error)
	GetCur() *pb.Blueprint
}

func LogErrors(mgr *pb.Manager) {
	errs := mgr.GetErrors()
	founderrs := false
	for id, e := range errs {
		if !founderrs {
			glog.Infoln("Printing connection errors.")
		}
		glog.Infof("id %d: error %v\n", id, e)
		founderrs = true
	}
	if !founderrs {
		glog.Infoln("No connection errors.")
	}

	return
}