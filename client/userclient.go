package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	bp "github.com/relab/smartMerge/blueprints"
	conf "github.com/relab/smartMerge/confProvider"
	"github.com/relab/smartMerge/elog"
	e "github.com/relab/smartMerge/elog/event"
	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/util"
)

func usermain() {
	flag.Parse()
	addrs, ids := util.GetProcs(*confFile, true)

	//Build initial blueprint.
	if *initsize > len(ids) {
		fmt.Fprintln(os.Stderr, "Not enough servers to fulfill initsize.")
		return
	}

	initBlp := new(bp.Blueprint)
	initBlp.Nodes = make([]*bp.Node, 0, len(ids))
	for i, id := range ids {
		if i >= *initsize {
			break
		}
		initBlp.Nodes = append(initBlp.Nodes, &bp.Node{Id: id})
	}
	initBlp.FaultTolerance = uint32(15)

	cp, mgr, err := NewConfP(addrs, *cprov, (*clientid))
	if err != nil {
		fmt.Println("Error creating confProvider: ", err)
		return
	}
	client, err := NewClient(initBlp, *alg, *opt, *clientid, cp)
	defer PrintErrors(mgr)
	if err != nil {
		fmt.Println("Error creating client: ", err)
		return
	}

	if *doelog {
		elog.Enable()
		defer elog.Flush()
	}

	for {
		fmt.Println("Choose operation:")
		fmt.Println("  1: Read")
		fmt.Println("  2: Write")
		fmt.Println("  3: Regular Read")
		fmt.Println("  4: Reconfigure")
		fmt.Println("  5: BenchmarkWrites")
		fmt.Println("  0: Exit")

		var op int
		_, err := fmt.Scanf("%d", &op)
		if err != nil {
			fmt.Println("invalid input.")
			continue
		}

		switch op {
		case 1:
			reqsent := time.Now()
			bytes, cnt := client.Read(cp)
			elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
			state := string(bytes)
			fmt.Println("Current value is: ", state)
			fmt.Printf("Has %d bytes.\n", len(bytes))
			fmt.Printf("Did %d accesses.\n", cnt)
		case 2:
			var str string
			fmt.Print("Insert string to write: ")
			fmt.Scanln(&str)
			reqsent := time.Now()
			cnt := client.Write(cp, []byte(str))
			elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
			fmt.Printf("Did %d accesses.\n", cnt)
		case 3:
			reqsent := time.Now()
			bytes, cnt := client.RRead(cp)
			elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
			state := string(bytes)
			fmt.Println("Current value is: ", state)
			fmt.Printf("Has %d bytes.\n", len(bytes))
			fmt.Printf("Did %d accesses.\n", cnt)
		case 4:
			handleReconf(client, cp, ids)
		case 5:
			var size, writes int
			fmt.Println("Enter size:")
			fmt.Scanln(&size)
			fmt.Println("Enter writes:")
			fmt.Scanln(&writes)
			doWrites(client, cp, size, writes, nil)
		default:
			return
		}
	}

}

func handleReconf(c RWRer, cp conf.Provider, ids []uint32) {
	cur := c.GetCur()
	fmt.Println("Current Blueprint is: ", cur.Nodes)
	fmt.Println("Type 1 or 2 for add or remove?")
	fmt.Println("  1: Add")
	fmt.Println("  2: Remove")

	var adrem int
	_, err := fmt.Scanf("%d", &adrem)
	switch adrem {
	case 1:
		fmt.Println("Available ids:")
		for _, id := range ids {
			fmt.Println(id)
		}
		fmt.Println("Type the id for the process to be added.")
		var id uint32
		_, err = fmt.Scanf("%d", &id)
		if err != nil {
			fmt.Println(err)
			return
		}

		target := cur.Copy()

		if !target.Add(id) {
			fmt.Printf("Node wit id %d was already added.\n", id)
			return
		}

		fmt.Println("Starting reconfiguration with target ", target.Nodes)
		reqsent := time.Now()
		cnt, err := c.Reconf(cp, target)
		elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
		if err != nil {
			fmt.Println("Reconf returned error: ", err)
		}
		fmt.Printf("did %d accesses.\n", cnt)
		fmt.Println("new blueprint is ", c.GetCur().Nodes)
		return
	case 2:
		fmt.Println("Ids in the current configuration:")
		for _, id := range cur.Ids() {
			fmt.Println(id)
		}
		fmt.Println("Type the id to be removed.")
		var id uint32
		_, err = fmt.Scanf("%d", &id)
		if err != nil {
			fmt.Println(err)
			return
		}

		target := cur.Copy()
		if !target.Rem(uint32(id)) {
			fmt.Println("Node is not part of current configuration.")
			return
		}

		reqsent := time.Now()
		cnt, err := c.Reconf(cp, target)
		elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))

		if err != nil {
			fmt.Println("Reconf returned error: ", err)
		}

		fmt.Printf("did %d accesses.\n", cnt)
		fmt.Println("new blueprint is ", c.GetCur().Nodes)
		return
	default:
		return
	}

}

func PrintErrors(mgr *pb.Manager) {
	founderrs := false
	for _, n := range mgr.Nodes() {
		if err := n.LastErr(); err != nil {
			if !founderrs {
				fmt.Println("Printing connection errors.")
			}
			fmt.Printf("id %d: error %v\n", n.ID(), err)
			founderrs = true
		}
	}
	if !founderrs {
		fmt.Println("No connection errors.")
	}

	return
}
