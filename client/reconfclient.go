package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
	conf "github.com/relab/smartMerge/confProvider"
	"github.com/relab/smartMerge/elog"
	e "github.com/relab/smartMerge/elog/event"
	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/util"
)

func expmain() {
	parseFlags()

	addrs, ids := util.GetProcs(*confFile, false)

	//Build initial blueprint.
	if *initsize > len(ids) && *initsize < 100 {
		glog.Errorln("Not enough servers to fulfill initsize.")
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

	if *doelog {
		elog.Enable()
		defer elog.Flush()
	}

	var wg sync.WaitGroup
	syncchan := make(chan struct{})

	for i := 0; i < *nclients; i++ {
		glog.Infoln("starting client number: ", i)
		cp, mgr, err := NewConfP(addrs, *cprov, (*clientid)+i)
		if err != nil {
			glog.Errorln("Error creating confProvider: ", err)
			continue
		}
		cl, err := NewClient(initBlp, *alg, *opt, (*clientid)+i, cp)
		if err != nil {
			glog.Errorln("Error creating client: ", err)
			continue
		}

		if *useleader {
			if *alg == "sm" || *alg == "" {
				cl, err = createForwarder(cl, mgr, ids[len(ids)-1])
				if err != nil {
					glog.Errorln("Error creating forwarder:", err)
					continue
				}
			} else {
				glog.Errorln("Can not create forwarder for algorithm ", *alg)
			}
		}

		defer PrintErrors(mgr)
		wg.Add(1)
		switch {
		case *cont:
			if i%2 == 0 {
				go contremove(cl, cp, ids, syncchan, (*clientid)+(i/2), &wg)
			} else {
				go contadd(cl, cp, ids, syncchan, (*clientid)+(i/2), &wg)
			}
		case *rm:
			go remove(cl, cp, ids, syncchan, (*clientid)+i, &wg)
		case *add:
			go adds(cl, cp, ids, syncchan, *initsize+i, &wg)
		case *repl:
			go replace(cl, cp, ids, syncchan, (*clientid)+i, &wg)
		}
	}

	if *cont {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	loopSignals:
		for {
			select {
			case signal := <-signalChan:
				if exit := handleSignal(signal); exit {
					glog.Infoln("stopping goroutines")
					close(syncchan)
					break loopSignals //break for loop, not only select
				}
			}
		}
	} else {
		time.Sleep(1 * time.Second)
		close(syncchan)
	}

	glog.Infoln("waiting for goroutines")
	wg.Wait()
	glog.Infoln("finished waiting")
	return

}

func contremove(c RWRer, cp conf.Provider, ids []uint32, sc chan struct{}, i int, wg *sync.WaitGroup) {
	if len(ids) <= i {
		glog.Errorf("Configuration file does not hold %d processes.\n", i+1)
		return
	}

	defer wg.Done()
	for {
		target := c.GetCur(cp) //GetCur returns a copy, not the real thing.
		if !target.Rem(ids[i]) {
			glog.Infoln("Could not remove %v\n.", ids[i])
		} else {
			reqsent := time.Now()
			cnt, err := c.Reconf(cp, target)
			if err == nil || cnt == 0 {
				elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
			} else {
				glog.Errorln("Reconf returned error:", err)
			}
		}

		select {
		case <-sc:
			return
		default:
			continue
		}
	}
}

func contadd(c RWRer, cp conf.Provider, ids []uint32, sc chan struct{}, i int, wg *sync.WaitGroup) {
	if len(ids) <= i {
		glog.Errorf("Configuration file does not hold %d processes.\n", i+1)
		return
	}

	defer wg.Done()
	for {
		target := c.GetCur(cp) //GetCur returns a copy, not the real thing.
		if !target.Add(ids[i]) {
			glog.V(4).Infoln("Could not add %v\n.", ids[i])
		} else {
			reqsent := time.Now()
			cnt, err := c.Reconf(cp, target)
			if err == nil || cnt == 0 {
				elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
			} else {
				glog.Errorln("Reconf returned error:", err)
			}
		}

		select {
		case <-sc:
			return
		default:
			continue
		}
	}
}

func replace(c RWRer, cp conf.Provider, ids []uint32, sc chan struct{}, i int, wg *sync.WaitGroup) {
	defer wg.Done()
	cur := c.GetCur(cp)
	if len(ids) <= *initsize+i {
		glog.Errorf("Configuration file does not hold %d processes.\n", *initsize+i+1)
		return
	}
	target := cur.Copy()
	if !target.Rem(ids[i]) {
		glog.Errorln("Remove did not result in new blueprint.")
	}
	target.Add(ids[*initsize+i])

	<-sc
	reqsent := time.Now()
	cnt, err := c.Reconf(cp, target)
	elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))

	if err != nil {
		glog.Errorln("Reconf returned error: ", err)
	}
	return
}

func remove(c RWRer, cp conf.Provider, ids []uint32, sc chan struct{}, i int, wg *sync.WaitGroup) {
	defer wg.Done()
	cur := c.GetCur(cp)
	if len(ids) <= i {
		glog.Errorf("Configuration file does not hold %d processes.\n", i+1)
		return
	}
	target := cur.Copy()
	if !target.Rem(ids[i]) {
		glog.Errorln("Remove did not result in new blueprint.")
	}

	<-sc
	reqsent := time.Now()
	cnt, err := c.Reconf(cp, target)
	elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))

	if err != nil {
		glog.Errorln("Reconf returned error: ", err)
	}
	return
}

func adds(c RWRer, cp conf.Provider, ids []uint32, sc chan struct{}, i int, wg *sync.WaitGroup) {
	defer wg.Done()
	cur := c.GetCur(cp)
	if len(ids) <= i {
		glog.Errorf("Configuration file does not hold %d processes.\n", i+1)
		return
	}
	target := cur.Copy()
	target.Add(ids[i])

	if target.Equals(cur) {
		glog.Errorln("Add did not result in a new configuration.")
	}

	<-sc

	reqsent := time.Now()
	cnt, err := c.Reconf(cp, target)
	elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))

	if err != nil {
		glog.Errorln("Reconf returned error: ", err)
	}
	return
}

func createForwarder(cl RWRer, mgr *pb.Manager, lid uint32) (RWRer, error) {
	ids := mgr.ToIds([]uint32{lid})
	cnf, err := mgr.NewConfiguration(ids, 1, conf.ConfTimeout)
	if err != nil {
		return nil, err
	}
	return &FwdClient{cl, cnf}, nil
}
