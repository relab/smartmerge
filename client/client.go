package main

import (
	"flag"
	"fmt"
	
	"github.com/relab/smartMerge/util"
	"github.com/relab/smartMerge/rpc"
	lat "github.com/relab/smartMerge/directCombineLattice"
	"github.com/relab/smartMerge/smclient"
)

var( 
	confFile = flag.String("conf","config", "the config file, a list of host:port addresses.")
	clientid = flag.Int("id", 0, "the client id")
)

func main() {
	flag.Parse()
	addrs, ids := util.GetProcs(*confFile, true)
	
	initBlp := lat.Blueprint{Add: map[lat.ID]bool{lat.ID(ids[0]): true}, Rem: nil}
	
	mgr,err := rpc.NewManager(addrs)
	if err != nil {
		fmt.Println("Creating manager returned error: ", err)
		return
	}
	
	defer PrintErrors(mgr)
	
	client, err := smclient.New(&initBlp, mgr, uint32(*clientid))
	if err != nil {
		fmt.Println("Creating client returned error: ",err)
		return
	}
	
	for {
		fmt.Println("Choose operation:")
		fmt.Println("  1: Read")
		fmt.Println("  2: Write")
		fmt.Println("  3: Reconfigure")
		fmt.Println("  0: Exit")
		
		var op int
		_, err := fmt.Scanf("%d",&op)
		if err != nil {
			fmt.Println("invalid input.")
			continue
		}
		
		switch op {
		case 1:
			bytes := client.Read()
			state := string(bytes)
			fmt.Println("Current value is: ", state)
		case 2:
			var str string
			fmt.Print("Insert string to write: ")
			fmt.Scanln(&str)
			client.Write([]byte(str))
		case 3: 
			handleReconf(client, ids)
		default: 
			return
		}
	}
	
	
}

func handleReconf(c *smclient.SmClient, ids []uint32) {
	fmt.Println("Current Blueprint is: ", c.Blueps[0])
	fmt.Println("Type 1 or 2 for add or remove?")
	fmt.Println("  1: Add")
	fmt.Println("  2: Remove")
	
	var adrem int
	_, err := fmt.Scanf("%d", &adrem)
	switch adrem {
	case 1:
		fmt.Println("Available ids:")
		for _,id := range ids {
			fmt.Println(id)
		}
		fmt.Println("Type the id for the process to be added.")
		var id uint32
		_,err = fmt.Scanf("%d", &id)
		if err != nil {
			fmt.Println(err)
			return
		}
		lid := lat.ID(id)
		if _,ok := c.Blueps[0].Rem[lid]; ok {
			return
		}
		if _,ok := c.Blueps[0].Add[lid]; ok {
			return
		}
		
		target := new(lat.Blueprint)
		for k,_ := range c.Blueps[0].Add {
			target.AddP(k)
		}
		for k,_ := range c.Blueps[0].Rem {
			target.RemP(k)
		}
		target.AddP(lid)
		fmt.Println("Starting reconfiguration with target ", target)
		err = c.Reconf(target)
		if err != nil {
			fmt.Println("Reconf returned error: ", err)
		}
	case 2:
		fmt.Println("Ids in the current configuration:")
		for id := range c.Blueps[0].Add {
			fmt.Println(id)
		}
		fmt.Println("Type the id to be removed.")
		var id uint32
		_,err = fmt.Scanf("%d", &id)
		if err != nil {
			fmt.Println(err)
			return
		}
		
		lid := lat.ID(id)
		if _,ok := c.Blueps[0].Rem[lid]; ok {
			return
		}
		if _,ok := c.Blueps[0].Add[lid]; !ok {
			return
		}
		
		target := new(lat.Blueprint)
		target.RemP(lid)
		target = target.Merge(c.Blueps[0])
		err = c.Reconf(target)
		
		if err != nil {
			fmt.Println("Reconf returned error: ", err)
		}
		
		fmt.Println("new blueprint is ", c.Blueps[0])
		return
	default: 
		return
	}

}

func PrintErrors(mgr *rpc.Manager) {
	errs := mgr.GetErrors()
	for id, e := range errs {
		fmt.Printf("id %d: error %v\n", id, e)
	}
	return
}