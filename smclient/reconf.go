package smclient

import (
	"errors"
	"fmt"

	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) Reconf(prop *pb.Blueprint) (cnt int, err error) {
	prop, cnt = smc.lagree(prop)
	//fmt.Printf("LA returned Blueprint with %d procs and %d removals.\n", len(prop.Add), len(prop.Rem))

	//Proposed blueprint is already in place, or outdated.
	if prop.LearnedCompare(smc.Blueps[0]) == 0 {
		return cnt, nil
	}

	if prop.Compare(smc.Blueps[0]) == 1 {
		return cnt, nil
	}

	if prop.LearnedCompare(smc.Blueps[len(smc.Blueps)-1]) == 1 {
		prop = smc.Blueps[len(smc.Blueps)-1]
	}

	if len(prop.Add) == 0 {
		return cnt, errors.New("Abort before proposing unacceptable configuration.")
	}

	cur := 0
	las := new(pb.Blueprint)
	rst := new(pb.State)
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		writeN, err := smc.Confs[i].AWriteN(&pb.AdvWriteN{uint32(smc.Blueps[i].Len()),prop})
		cnt++
		if err != nil {
			//Should log this for debugging
			fmt.Println("AWriteN returned error: ",err)
			panic("Error from AWriteN")
		}

		cur = smc.handleNewCur(cur, writeN.Reply.GetCur())
		smc.handleNext(i, writeN.Reply.GetNext())
		las = las.Merge(writeN.Reply.GetLAState())
		if rst.Compare(writeN.Reply.GetState()) == 1 {
			rst = writeN.Reply.GetState()
		}

		prop = smc.Blueps[len(smc.Blueps)-1]
		//fmt.Println("Len Blueps, Confs: ", len(smc.Blueps), len(smc.Confs))
		//fmt.Println("Cur has index ", cur)
	}

	if i := len(smc.Confs) - 1; i > cur {
		setS, err := smc.Confs[i].SetState(&pb.NewState{CurC: uint32(smc.Blueps[i].Len()), Cur: smc.Blueps[i], State: rst, LAState: las})
		cnt++
		if err != nil {
			//Not sure what to do:
			fmt.Println("SetState returned error, not sure what to do")
			panic("Error from SetState")
		}
		
		cur = smc.handleNewCur(i, setS.Reply.GetCur())
	}

	smc.Blueps = smc.Blueps[cur:]
	smc.Confs = smc.Confs[cur:]

	return cnt, nil
}

func (smc *SmClient) lagree(prop *pb.Blueprint) (*pb.Blueprint, int) {
	cnt := 0
	cur := 0
	prop = prop.Merge(smc.Blueps[0])
	for i := 0; i < len(smc.Confs); i++ {
		//fmt.Println("LA in conf ", smc.Blueps[0], " with index ", i)
		if i < cur {
			continue
		}

		laProp, err := smc.Confs[i].LAProp(&pb.LAProposal{uint32(smc.Blueps[i].Len()), prop})
		cnt++
		if err != nil {
			fmt.Println("LA prop returned error: ", err)
			panic("Error from LAProp")
		}

		cur = smc.handleNewCur(cur, laProp.Reply.GetCur())
		la := laProp.Reply.GetLAState()
		if la != nil && !prop.Equals(la) {
			//fmt.Println("LA prop returned new LA state ", la)
			prop = la
			i--
			continue
		}

		smc.handleNext(i, laProp.Reply.GetNext())
	}
	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return prop, cnt
}
