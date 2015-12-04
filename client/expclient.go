package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/relab/goxos/kvs/bgen"
	conf "github.com/relab/smartMerge/confProvider"
	cc "github.com/relab/smartMerge/consclient"
	"github.com/relab/smartMerge/doreconf"
	dyna "github.com/relab/smartMerge/dynaclient"
	"github.com/relab/smartMerge/elog"
	e "github.com/relab/smartMerge/elog/event"
	pb "github.com/relab/smartMerge/proto"
	qf "github.com/relab/smartMerge/qfuncs"
	smc "github.com/relab/smartMerge/smclient"
	ssr "github.com/relab/smartMerge/ssrclient"
	"github.com/relab/smartMerge/util"
	grpc "google.golang.org/grpc"
)

var (
	//General
	gcOff      = flag.Bool("gc-off", false, "turn garbage collection off")
	showHelp   = flag.Bool("help", false, "show this help message and exit")
	allCores   = flag.Bool("all-cores", false, "use all available logical CPUs")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to this file")
	httpprof   = flag.Bool("httpprof", false, "enable profiling via http server")

	// Mode
	mode   = flag.String("mode", "", "run mode: (user | bench | exp )")
	alg    = flag.String("alg", "", "algorithm to be used: (sm | dyna | ssr | cons )")
	opt    = flag.String("opt", "", "which optimization to use: ( no | doreconf )")
	cprov  = flag.String("cprov", "normal", "which configuration provider: (normal | thrifty | norecontact ) ")
	doelog = flag.Bool("elog", false, "log latencies in user or exp mode.")

	//Config
	confFile  = flag.String("conf", "config", "the config file, a list of host:port addresses.")
	clientid  = flag.Int("id", 0, "the client id")
	nclients  = flag.Int("nclients", 1, "the number of clients")
	initsize  = flag.Int("initsize", 1, "the number of servers in the initial configuration")
	useleader = flag.Bool("useleader", false, "let a leader handle reconfigurations.")

	//Read or Write Bench
	contW  = flag.Bool("contW", false, "continuously write")
	contR  = flag.Bool("contR", false, "continuously read")
	reads  = flag.Int("reads", 0, "number of reads to be performed.")
	writes = flag.Int("writes", 0, "number of writes to be performed.")
	size   = flag.Int("size", 16, "number of bytes for value.")
	regul  = flag.Bool("regular", false, "do only regular reads")

	//Reconf Exp
	rm   = flag.Bool("rm", false, "remove nclients servers concurrently.")
	add  = flag.Bool("add", false, "add nclients servers concurrently")
	repl = flag.Bool("repl", false, "replace nclient many servers concurrently")
	cont = flag.Bool("cont", false, "continuously reconfigure")
	logT = flag.Bool("logThroughput", false, "Log reads per second.")
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
	} else {
		runtime.GOMAXPROCS(1)
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
	initBlp.Nodes = make([]*pb.Node, 0, len(ids))
	for i, id := range ids {
		if i >= *initsize {
			break
		}
		initBlp.Nodes = append(initBlp.Nodes, &pb.Node{Id: id})
	}
	initBlp.FaultTolerance = uint32(15)

	checkFlags(*alg, *cprov, *opt)

	var wg sync.WaitGroup

	elog.Enable()
	defer elog.Flush()
	stop := make(chan struct{}, *nclients)

	for i := 0; i < *nclients; i++ {
		glog.Infof("starting configProvider and manager %d at time %v\n", i, time.Now())
		cp, mgr, err := NewConfP(addrs, *cprov, (*clientid)+i)
		if err != nil {
			glog.Errorln("Error creating confProvider: ", err)
			continue
		}

		defer PrintErrors(mgr)
		glog.Infoln("starting client with id", (*clientid)+i)
		cl, err := NewClient(initBlp, *alg, *opt, (*clientid)+i, cp)
		if err != nil {
			glog.Errorln("Error creating client: ", err)
			continue
		}

		wg.Add(1)
		switch {
		case *contW:
			go contWrite(cl, cp, *size, stop, &wg)
		case *contR:
			go contRead(cl, cp, stop, *regul, *logT, &wg)
		case *reads > 0:
			go doReads(cl, cp, *reads, *regul, &wg)
		case *writes > 0:
			go doWrites(cl, cp, *size, *writes, &wg)
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

func NewConfP(addrs []string, cprov string, id int) (cp conf.Provider, mgr *pb.Manager, err error) {
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
		pb.WithGetPromiseQuorumFunc(qf.GetPromiseQF),
		pb.WithAcceptQuorumFunc(qf.AcceptQF),
		pb.WithDWriteNQuorumFunc(qf.DWriteNQF),
		pb.WithDSetStateQuorumFunc(qf.DSetStateQF),
		pb.WithDWriteNSetQuorumFunc(qf.DWriteNSetQF),
		pb.WithDSetCurQuorumFunc(qf.DSetCurQF),
		pb.WithGetOneNQuorumFunc(qf.GetOneNQF),
		pb.WithSpSnOneQuorumFunc(qf.SpSnOneQF),
		pb.WithSCommitQuorumFunc(qf.SCommitQF),
		pb.WithSSetStateQuorumFunc(qf.SSetStateQF),
	)
	if err != nil {
		glog.Errorln("Creating manager returned error: ", err)
		return
	}

	cp = conf.NewProvider(mgr, id)
	switch cprov {
	case "norecontact":
		break
	case "thrifty":
		cp = &conf.ThriftyConfP{cp}
	case "normal", "":
		cp = &conf.NormalConfP{cp}
	default:
		glog.Fatalf("confprovider %v is not supported.\n", cprov)
	}
	return
}

func NewClient(initB *pb.Blueprint, alg string, opt string, id int, cp conf.Provider) (cl RWRer, err error) {
	switch alg {
	case "", "sm":
		switch opt {
		case "", "no":
			cl, err = smc.New(initB, uint32(id), cp)
		case "doreconf":
			cl, err = doreconf.NewSM(initB, uint32(id), cp)
		default:
			glog.Fatalf("optimization %v not supported.\n", opt)
		}
	case "dyna":
		cl, err = dyna.New(initB, uint32(id), cp)
	case "ssr":
		cl, err = ssr.New(initB, uint32(id), cp)
	case "cons":
		switch opt {
		case "", "no":
			cl, err = cc.New(initB, uint32(id), cp)
		case "doreconf":
			cl, err = doreconf.NewCons(initB, uint32(id), cp)
		default:
			glog.Fatalln("optimization recontact not yet supported.")
		}
	default:
		glog.Fatalln("this algorithm is not supported.")
	}
	return
}

func contWrite(cl RWRer, cp conf.Provider, size int, stop chan struct{}, wg *sync.WaitGroup) {
	glog.Infoln("starting continous write")
	var (
		value   = make([]byte, size)
		cnt     int
		reqsent time.Time
	)

	bgen.GetBytes(value)

loop:
	for {
		reqsent = time.Now()
		cnt = cl.Write(cp, value)
		elog.Log(e.NewTimedEventWithMetric(e.ClientWriteLatency, reqsent, uint64(cnt)))
		if cnt > 100 {
			break
		}
		select {
		case <-stop:
			break loop
		default:
			//Continue
		}
	}
	glog.Infoln("finished continous write")
	wg.Done()
}

func contRead(cl RWRer, cp conf.Provider, stop chan struct{}, reg bool, logT bool, wg *sync.WaitGroup) {
	glog.Infoln("starting continous read")
	var (
		c        int
		cnt      int
		reqsent  time.Time
		throuput uint64
	)

	cchan := make(chan int, 1)
	var tick <-chan time.Time

	if logT {
		ts := time.Now().Truncate(time.Second).Add(time.Second)
		time.Sleep(ts.Sub(time.Now()))
		tick = time.Tick(1 * time.Second)
	}

loop:
	for {
		reqsent = time.Now()
		go func() {
			if reg {
				_, c = cl.RRead(cp)
			} else {
				_, c = cl.Read(cp)
			}
			cchan <- c
		}()
	select_:
		select {
		case cnt = <-cchan:
			throuput++
			elog.Log(e.NewTimedEventWithMetric(e.ClientReadLatency, reqsent, uint64(cnt)))
		case <-tick:
			elog.Log(e.NewEventWithMetric(e.ThroughputSample, throuput))
			throuput = 0
			goto select_
		}

		select {
		case <-stop:
			glog.Infoln("received stopping signal")
			break loop
		default:
			// Continue
		}
	}
	glog.Infoln("finished continous read")
	wg.Done()
}

func doWrites(cl RWRer, cp conf.Provider, size int, writes int, wg *sync.WaitGroup) {
	var (
		value   = make([]byte, size)
		cnt     int
		reqsent time.Time
	)

	bgen.GetBytes(value)
	for i := 0; i < writes; i++ {
		reqsent = time.Now()
		cnt = cl.Write(cp, value)
		elog.Log(e.NewTimedEventWithMetric(e.ClientWriteLatency, reqsent, uint64(cnt)))
	}
	glog.Infoln("finished writes")
	if wg != nil {
		wg.Done()
	}
}

func doReads(cl RWRer, cp conf.Provider, reads int, reg bool, wg *sync.WaitGroup) {
	var (
		cnt     int
		reqsent time.Time
	)

	for i := 0; i < reads; i++ {
		reqsent = time.Now()
		if reg {
			_, cnt = cl.RRead(cp)
		} else {
			_, cnt = cl.Read(cp)
		}
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
	RRead(conf.Provider) ([]byte, int)
	Read(conf.Provider) ([]byte, int)
	Write(cp conf.Provider, val []byte) int
	Reconf(cp conf.Provider, prop *pb.Blueprint) (int, error)
	GetCur(conf.Provider) *pb.Blueprint
}

func LogErrors(mgr *pb.Manager) {
	errs := mgr.GetErrors()
	founderrs := false
	for id, e := range errs {
		if !founderrs {
			glog.Errorln("Printing connection errors.")
		}
		glog.Errorf("id %d: error %v\n", id, e)
		founderrs = true
	}
	if !founderrs {
		glog.Infoln("No connection errors.")
	}

	return
}

func checkFlags(alg, cprov, opt string) {
	if alg == "cons" && cprov == "norecontact" && opt == "doreconf" {
		glog.Errorln("Unsupported flag combination. With alg=cons and doreconf, norecontact will result in no benefit.")
	} else if alg == "cons" && cprov == "norecontact" {
		glog.Warningln("To use norecontact with consensus, the servers have to use alg=sm, not alg=cons")
	}
	if alg == "dyna" {
		if opt == "doreconf" {
			glog.Warningln("Doreconf is default for Dynastore algorithm.")
		}
		if cprov == "norecontact" {
			glog.Warningln("Norecontact not supported in Dynastore algorithm.")
		}
	}
	if alg == "ssr" {
		if opt == "doreconf" {
			glog.Warningln("Doreconf is default for the speculating snapshot register.")
		}
		if cprov == "norecontact" {
			glog.Warningln("Norecontact not supported for speculated snapshot register.")
		}
	}
}
