package main

import (
	"flag"
	//"strconv"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/golang/glog"
	conf "github.com/relab/smartMerge/confProvider"
	"github.com/relab/smartMerge/leader"
	pb "github.com/relab/smartMerge/proto"
	qf "github.com/relab/smartMerge/qfuncs"
	"github.com/relab/smartMerge/regserver"
	"github.com/relab/smartMerge/util"
	grpc "google.golang.org/grpc"
)

var (
	port     = flag.Int("port", 10000, "this servers address ip:port.")
	gcoff    = flag.Bool("gcoff", false, "turn garbage collection off.")
	alg      = flag.String("alg", "", "algorithm to use (sm | dyna | ssr | cons )")
	allCores = flag.Bool("all-cores", false, "use all available logical CPUs")

	noabort = flag.Bool("no-abort", false, "do not send aborting new-cur information.")

	cprov = flag.String("cprov", "normal", "which configuration provider: (normal | thrifty | norecontact ) ")
	//Config
	confFile = flag.String("conf", "config", "the config file, a list of host:port addresses.")
	clientid = flag.Int("id", 0, "the client id")
	initsize = flag.Int("initsize", 1, "the number of servers in the initial configuration")
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

	//Parse Processes from Config file.
	addrs, ids := util.GetProcs(*confFile, false)

	//Build initial blueprint.
	if *initsize > len(ids) && *initsize < 100 {
		glog.Errorln(os.Stderr, "Not enough servers to fulfill initsize.")
		return
	}

	var err error
	var rs *regserver.ConsServer
	glog.Infoln("Starting Server with port: ", *port)
	switch *alg {
	case "", "sm":
		_, err = regserver.StartAdv(*port, *noabort)
	case "dyna":
		_, err = regserver.StartDyna(*port)
	case "ssr":
		_, err = regserver.StartSSR(*port)
	case "cons":
		rs, err = regserver.StartCons(*port, *noabort)
	}

	if err != nil {
		glog.Fatalln("Starting server returned error", err)
	} else {

		time.Sleep(1 * time.Second) //Better than a long timeout here is a long timeout for trying to connect.

		initBlp := new(pb.Blueprint)
		initBlp.Nodes = make([]*pb.Node, 0, len(ids))
		for i, id := range ids {
			if i >= *initsize {
				break
			}
			initBlp.Nodes = append(initBlp.Nodes, &pb.Node{Id: id})
		}
		initBlp.FaultTolerance = uint32(15)

		glog.Infof("starting configProvider and manager at time %v\n", time.Now())
		cp, mgr, err := NewConfP(addrs, *cprov, (*clientid))
		if err != nil {
			glog.Errorln("Error creating confProvider: ", err)
			return
		}

		defer LogErrors(mgr)
		glog.Infoln("starting client with id", (*clientid))
		l, err := leader.New(initBlp, uint32(*clientid), cp)
		if err != nil {
			glog.Errorln("Error creating leader: ", err)
			return
		}

		glog.Infoln("starting to run")
		l.Run()
		defer l.Stop()

		rs.AddLeader(l)
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

func NewConfP(addrs []string, cprov string, id int) (cp conf.Provider, mgr *pb.Manager, err error) {
	mgr, err = pb.NewManager(addrs, pb.WithGrpcDialOptions(
		grpc.WithBlock(),
		grpc.WithTimeout(6000*time.Millisecond),
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
