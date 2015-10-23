package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	"sync"

	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/elog"
	e "github.com/relab/smartMerge/elog/event"
	"github.com/relab/smartMerge/util"
)

func expmain() {
	flag.Parse()
	addrs, ids := util.GetProcs(*confFile, true)

	//Build initial blueprint.
	if *initsize > len(ids) && *initsize < 100 {
		fmt.Fprintln(os.Stderr, "Not enough servers to fulfill initsize.")
		return
	}

	if *initsize > 100 {
		initBlp := &pb.Blueprint{Add: ids, Rem: nil}
	} else {
		initBlp := &pb.Blueprint{Add: ids[:*initsize], Rem: nil}
	}

	if *doelog {
		elog.Enable()
		defer elog.Flush()
	}

	var wg sync.WaitGroup
	syncchan := make(chan struct{})

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
		case *rm:
			if *clientid + *nclients >= *initsize {
				fmt.Println("Not enough processes to remove.")
				return
			}
			go remove(cl, ids, syncchan, (*clientid)+i, &wg)
		case *add:
			go adds(cl, ids, syncchan, *initsize + i, &wg)
		}
	}
	time.Sleep(2 * time.Second)
	close(syncchan)

	fmt.Println("waiting for goroutines")
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
		fmt.Println("Reconf returned error: ", err)
	}
	return
}

func adds(c RWRer,ids []uint32, sc chan struct{}, i int, wg *sync.WaitGroup) {
	defer wg.Done()	
	cur := c.GetCur()
	if len(ids)<= i {
		fmt.Printf("Configuration file does not hold %d processes.", i+1)
		return
	}
	target := new(pb.Blueprint)
	target.Add = []uint32{ids[i]}
	target = target.Merge(cur)
	
	if target.Equals(cur) {
		fmt.Println("Add did not result in a new configuration.")
	}
	
	<-sc
	
	reqsent := time.Now()
	cnt, err := c.Reconf(target)
	elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))

	if err != nil {
		fmt.Println("Reconf returned error: ", err)
	}
	return
}
