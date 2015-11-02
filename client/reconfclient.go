package main

import (
	"flag"
	"os"
	"time"
	"sync"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/elog"
	e "github.com/relab/smartMerge/elog/event"
	"github.com/relab/smartMerge/util"
)

func expmain() {
	flag.Parse()
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

	if *doelog {
		elog.Enable()
		defer elog.Flush()
	}

	var wg sync.WaitGroup
	syncchan := make(chan struct{})

	for i := 0; i < *nclients; i++ {
		glog.Infoln("starting client number: ", i)
		cl, mgr, err := NewClient(addrs, initBlp, *alg, *opt, (*clientid)+i)
		if err != nil {
			glog.Errorln("Error creating client: ", err)
			continue
		}

		defer PrintErrors(mgr)
		wg.Add(1)
		switch {
		case *rm:
			if *clientid + *nclients >= *initsize {
				glog.Errorln("Not enough processes to remove.")
				return
			}
			go remove(cl, ids, syncchan, (*clientid)+i, &wg)
		case *add:
			go adds(cl, ids, syncchan, *initsize + i, &wg)
		}
	}
	time.Sleep(2 * time.Second)
	close(syncchan)

	glog.Infoln("waiting for goroutines")
	wg.Wait()
	time.Sleep(2 * time.Second)
	return
}

func remove(c RWRer,ids []uint32, sc chan struct{}, i int, wg *sync.WaitGroup) {
	defer wg.Done()	
	cur := c.GetCur()
	target := new(pb.Blueprint)
	target.Rem = []uint32{ids[i]}
	target = target.Merge(cur)
		
	<-sc
	reqsent := time.Now()
	cnt, err := c.Reconf(target)
	elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))

	if err != nil {
		glog.Errorln("Reconf returned error: ", err)
	}
	return
}

func adds(c RWRer,ids []uint32, sc chan struct{}, i int, wg *sync.WaitGroup) {
	defer wg.Done()	
	cur := c.GetCur()
	if len(ids)<= i {
		glog.Errorf("Configuration file does not hold %d processes.\n", i+1)
		return
	}
	target := new(pb.Blueprint)
	target.Add = []uint32{ids[i]}
	target = target.Merge(cur)
	
	if target.Equals(cur) {
		glog.Errorln("Add did not result in a new configuration.")
	}
	
	<-sc
	
	reqsent := time.Now()
	cnt, err := c.Reconf(target)
	elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))

	if err != nil {
		glog.Errorln("Reconf returned error: ", err)
	}
	return
}
