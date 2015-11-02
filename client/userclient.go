package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/relab/smartMerge/elog"
	e "github.com/relab/smartMerge/elog/event"
	"github.com/relab/smartMerge/util"
	pb "github.com/relab/smartMerge/proto"
)

func usermain() {
	flag.Parse()
	addrs, ids := util.GetProcs(*confFile, true)

	//Build initial blueprint.
	if *initsize > len(ids) {
		fmt.Fprintln(os.Stderr, "Not enough servers to fulfill initsize.")
		return
	}


	initBlp := pb.Blueprint{Add: ids[:*initsize], Rem: nil}

	client, mgr, err := NewClient(addrs, &initBlp, *alg, *opt, *clientid)
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
			bytes, cnt := client.Read()
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
			cnt := client.Write([]byte(str))
			elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
			fmt.Printf("Did %d accesses.\n", cnt)
		case 3:
			reqsent := time.Now()
			bytes, cnt := client.RRead()
			elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
			state := string(bytes)
			fmt.Println("Current value is: ", state)
			fmt.Printf("Has %d bytes.\n", len(bytes))
			fmt.Printf("Did %d accesses.\n", cnt)
		case 4:
			handleReconf(client, ids)
		case 5: 
			var size, writes int
			fmt.Println("Enter size:")
			fmt.Scanln(&size)
			fmt.Println("Enter writes:")
			fmt.Scanln(&writes)
			doWrites(client, size, writes, nil)
		default:
			return
		}
	}

}

func handleReconf(c RWRer, ids []uint32) {
	cur := c.GetCur()
	fmt.Println("Current Blueprint is: ", cur)
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
		for _, rid := range cur.Rem {
			if rid == id {
				fmt.Println("Id:", id, " was already removed.")
				return
			}
		}
		for _, aid := range cur.Add {
			if aid == id {
				fmt.Println("Id:", id, " was already added.")
				return
			}
		}

		target := new(pb.Blueprint)
		target.Add = []uint32{id}
		
		fmt.Println("Starting reconfiguration with target ", target)
		reqsent := time.Now()
		cnt, err := c.Reconf(target)
		elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))
		if err != nil {
			fmt.Println("Reconf returned error: ", err)
		}
		fmt.Printf("did %d accesses.\n", cnt)
		fmt.Println("new blueprint is ", c.GetCur())
		return
	case 2:
		fmt.Println("Ids in the current configuration:")
		for _, id := range cur.Add {
			fmt.Println(id)
		}
		fmt.Println("Type the id to be removed.")
		var id uint32
		_, err = fmt.Scanf("%d", &id)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, rid := range cur.Rem {
			if rid == id {
				fmt.Println("Id:", id, " was already removed.")
				return
			}
		}
		found := false
		for _, aid := range cur.Add {
			if aid == id {
				found = true
			}
		}
		if !found {
			fmt.Println("Id:", id, " was not added yet.")
			return
		}
		
		target := new(pb.Blueprint)
		target.Rem = []uint32{id}
		target = target.Merge(cur)

		reqsent := time.Now()
		cnt, err := c.Reconf(target)
		elog.Log(e.NewTimedEventWithMetric(e.ClientReconfLatency, reqsent, uint64(cnt)))

		if err != nil {
			fmt.Println("Reconf returned error: ", err)
		}

		fmt.Printf("did %d accesses.\n", cnt)
		fmt.Println("new blueprint is ", c.GetCur())
		return
	default:
		return
	}

}

func PrintErrors(mgr *pb.Manager) {
	errs := mgr.GetErrors()
	founderrs := false
	for id, e := range errs {
		if !founderrs {
			fmt.Println("Printing connection errors.")
		}
		fmt.Printf("id %d: error %v\n", id, e)
		founderrs = true
	}
	if !founderrs {
		fmt.Println("No connection errors.")
	}

	return
}
