package smclient

import (
	"errors"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) Reconf(prop *pb.Blueprint) (cnt int, err error) {
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(smc.Blueps[0]) == 1 {
		glog.V(3).Infof("C%d: Proposal is already in place.", smc.ID)
		return 0, nil
	}
	_, cnt, err = smc.reconf(prop, true, nil)
	return
}

func (smc *SmClient) reconf(prop *pb.Blueprint, regular bool, val []byte) (rst *pb.State, cnt int, err error) {
	if glog.V(6) {
		glog.Infof("C%d: Starting reconf\n", smc.ID)
	}
	
	if prop.Compare(smc.Blueps[0]) != 1 {
		prop, cnt, err = smc.lagree(prop)
		if err != nil {
			return nil, 0, err
		}
		if len(prop.Add) == 0 {
			glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
			return nil, cnt, errors.New("Abort before moving to unacceptable configuration.")
		}
	}
	
	cur := 0
	las := new(pb.Blueprint)
forconfiguration:
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}
		
		if prop.LearnedCompare(smc.Blueps[i]) != -1 {
			var st *pb.State
			st, cur, err = smc.doread(cur, i)
			if err != nil {
				return nil, 0, err
			}
			cnt++
			if rst.Compare(st) == 1 {
				rst = st
			}
		}
		
		if i < cur {
			continue forconfiguration
		}

		if len(smc.Blueps) > i + 1 {
			if prop.LearnedCompare(smc.Blueps[len(smc.Blueps)-1]) == 1{
				prop = smc.Blueps[len(smc.Blueps)-1]
			}
		}

		if prop.LearnedCompare(smc.Blueps[i]) == -1 {
			writeN, err := smc.Confs[i].AWriteN(&pb.AdvWriteN{uint32(smc.Blueps[i].Len()), prop})
			if glog.V(3) {
				glog.Infoln("AWriteN returned")
			}
			cnt++
			if err != nil {
				//Should log this for debugging
				glog.Errorln("AWriteN returned error: ", err)
				return nil, 0, err
			}

			cur = smc.handleOneCur(cur, writeN.Reply.GetCur())
			smc.handleNext(i, writeN.Reply.GetNext())
			las = las.Merge(writeN.Reply.GetLAState())
			if rst.Compare(writeN.Reply.GetState()) == 1 {
				rst = writeN.Reply.GetState()
			}

			prop = smc.Blueps[len(smc.Blueps)-1]
		}
	}

	if i := len(smc.Confs) - 1; i > cur || !regular {

		rst = smc.WriteValue(val, rst)
		setS, err := smc.Confs[i].SetState(&pb.NewState{CurC: uint32(smc.Blueps[i].Len()), Cur: smc.Blueps[i], State: rst, LAState: las})
		if glog.V(3) {
			glog.Infof("C%d: Set State in Configuration with length %d\n ",smc.ID, smc.Blueps[i].Len())
		}
		cnt++
		if err != nil {
			//Not sure what to do:
			glog.Errorf("C%d: SetState returned error, not sure what to do\n", smc.ID)
			return nil, 0, err
		}
		cur = smc.handleOneCur(i, setS.Reply.GetCur())
		smc.handleNext(i, setS.Reply.GetNext())
		if !regular && i+1 < len(smc.Confs) {
			goto forconfiguration
		}
	}

	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return rst, cnt, nil
}


// func (smc *SmClient) Reconf(prop *pb.Blueprint) (cnt int, err error) {
// 	if glog.V(2) {
// 		glog.Infoln("Starting reconfiguration")
// 	}
// 	prop, cnt = smc.lagree(prop)
// 	//fmt.Printf("LA returned Blueprint with %d procs and %d removals.\n", len(prop.Add), len(prop.Rem))
// 	if glog.V(3) {
// 		glog.Infof("Needed %d proposals to solv LA.", cnt)
// 	}
//
// 	//Proposed blueprint is already in place, or outdated.
// 	if prop.LearnedCompare(smc.Blueps[0]) == 0 {
// 		glog.V(3).Infoln("Proposal is already in place.")
// 		return cnt, nil
// 	}
//
// 	if prop.Compare(smc.Blueps[0]) == 1 {
// 		glog.V(3).Infoln("Proposal already outdated.")
// 		return cnt, nil
// 	}
//
// 	if prop.LearnedCompare(smc.Blueps[len(smc.Blueps)-1]) == 1 {
// 		prop = smc.Blueps[len(smc.Blueps)-1]
// 	}
//
// 	if len(prop.Add) == 0 {
// 		glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
// 		return 0, errors.New("Abort before proposing unacceptable configuration.")
// 	}
//
// 	cur := 0
// 	las := new(pb.Blueprint)
// 	rst := new(pb.State)
// 	for i := 0; i < len(smc.Confs); i++ {
// 		if i < cur {
// 			continue
// 		}
//
// 		writeN, err := smc.Confs[i].AWriteN(&pb.AdvWriteN{uint32(smc.Blueps[i].Len()), prop})
// 		if glog.V(3) {
// 			glog.Infoln("AWriteN returned")
// 		}
// 		cnt++
// 		if err != nil {
// 			//Should log this for debugging
// 			glog.Errorln("AWriteN returned error: ", err)
// 			return 0, err
// 		}
//
// 		cur = smc.handleOneCur(cur, writeN.Reply.GetCur())
// 		smc.handleNext(i, writeN.Reply.GetNext())
// 		las = las.Merge(writeN.Reply.GetLAState())
// 		if rst.Compare(writeN.Reply.GetState()) == 1 {
// 			rst = writeN.Reply.GetState()
// 		}
//
// 		prop = smc.Blueps[len(smc.Blueps)-1]
// 		//fmt.Println("Len Blueps, Confs: ", len(smc.Blueps), len(smc.Confs))
// 		//fmt.Println("Cur has index ", cur)
// 	}
//
// 	if i := len(smc.Confs) - 1; i > cur {
// 		setS, err := smc.Confs[i].SetState(&pb.NewState{CurC: uint32(smc.Blueps[i].Len()), Cur: smc.Blueps[i], State: rst, LAState: las})
// 		if glog.V(3) {
// 			glog.Infoln("Set State in Configuration with length: ", smc.Blueps[i].Len())
// 		}
// 		cnt++
// 		if err != nil {
// 			//Not sure what to do:
// 			glog.Errorln("SetState returned error, not sure what to do")
// 			return 0, err
// 		}
//
// 		cur = smc.handleOneCur(i, setS.Reply.GetCur())
// 	}
//
// 	smc.Blueps = smc.Blueps[cur:]
// 	smc.Confs = smc.Confs[cur:]
//
// 	return cnt, nil
// }

func (smc *SmClient) lagree(prop *pb.Blueprint) (dec *pb.Blueprint,cnt int, err error) {
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
			return nil, 0, err
		}
		if glog.V(4) {
			glog.Infoln("LAProp returned.")
		}

		cur = smc.handleOneCur(cur, laProp.Reply.GetCur())
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
	return prop, cnt, nil
}

func (smc *SmClient) handleOneCur(cur int, newCur *pb.Blueprint) int {
	if newCur == nil {
		return cur
	}
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Len(), smc.Blueps[cur].Len())
	}
	return smc.findorinsert(cur, newCur)
}

func (smc *SmClient) doread(curin, i int) (st *pb.State, cur int, err error) {
	read, errx := smc.Confs[i].AReadS(&pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[i].Len())})
	if errx != nil {
		glog.Errorf("C%d: error from AReadS: ", errx)
		return nil, 0, errx
		//return
	}
	if glog.V(6) {
		glog.Infof("C%d: AReadS returned with replies from \n",smc.ID, read.MachineIDs)
	}
	cur =smc.handleNewCur(curin, read.Reply.GetCur())

	smc.handleNext(i, read.Reply.GetNext())

	return read.Reply.GetState(), cur, nil
}

func (smc *SmClient) WriteValue(val []byte, st *pb.State) *pb.State {
	if val == nil {
		return st
	}
	return &pb.State{Value: val, Timestamp: st.Timestamp + 1, Writer:smc.ID}
}

