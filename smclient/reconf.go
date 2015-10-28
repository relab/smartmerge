package smclient

import (
	"errors"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) Reconf(prop *pb.Blueprint) (cnt int, err error) {
	if glog.V(2) {
		glog.Infoln("Starting reconfiguration")
	}
	prop, cnt = smc.lagree(prop)
	//fmt.Printf("LA returned Blueprint with %d procs and %d removals.\n", len(prop.Add), len(prop.Rem))

	//Proposed blueprint is already in place, or outdated.
	if prop.LearnedCompare(smc.Blueps[0]) == 0 {
		glog.V(3).Infoln("Proposal is already in place.")
		return cnt, nil
	}

	if prop.Compare(smc.Blueps[0]) == 1 {
		glog.V(3).Infoln("Proposal already outdated.")
		return cnt, nil
	}

	if prop.LearnedCompare(smc.Blueps[len(smc.Blueps)-1]) == 1 {
		prop = smc.Blueps[len(smc.Blueps)-1]
	}

	if len(prop.Add) == 0 {
		glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
		return 0, errors.New("Abort before proposing unacceptable configuration.")
	}

	cur := 0
	las := new(pb.Blueprint)
	rst := new(pb.State)
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		writeN, err := smc.Confs[i].AWriteN(&pb.AdvWriteN{uint32(smc.Blueps[i].Len()), prop})
		if glog.V(3) {
			glog.Infoln("AWriteN returned")
		}
		cnt++
		if err != nil {
			//Should log this for debugging
			glog.Errorln("AWriteN returned error: ", err)
			return 0, err
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
		if glog.V(3) {
			glog.Infoln("Set State in Configuration with length: ", smc.Blueps[i].Len())
		}
		cnt++
		if err != nil {
			//Not sure what to do:
			glog.Errorln("SetState returned error, not sure what to do")
			return 0, err
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
			glog.Errorln("LA prop returned error: ", err)
			panic("Error from LAProp")
		}
		if glog.V(4) {
			glog.Infoln("LAProp returned.")
		}

		cur = smc.handleNewCur(cur, laProp.Reply.GetCur())
		la := laProp.Reply.GetLAState()
		if la != nil && !prop.Equals(la) {
			if glog.V(3) {
				glog.Infoln("LAProp returned new state, try again.")
			}
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
