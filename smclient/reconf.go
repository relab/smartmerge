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
	
	if smc.doCons {
		_, cnt, err = smc.consreconf(prop, true, nil)
	} else {
		_, cnt, err = smc.reconf(prop, true, nil)
	}
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
		if len(prop.Ids()) == 0 {
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
			st, _, cur, err = smc.doread(cur, i)
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

		if len(smc.Blueps) > i+1 {
			if prop.LearnedCompare(smc.Blueps[len(smc.Blueps)-1]) == 1 {
				prop = smc.Blueps[len(smc.Blueps)-1]
			}
		}

		if prop.LearnedCompare(smc.Blueps[i]) == -1 {
			writeN, err := smc.Confs[i].AWriteN(&pb.WriteN{uint32(smc.Blueps[i].Len()), prop})
			if glog.V(3) {
				glog.Infof("C%d: AWriteN returned\n", smc.ID)
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

			if prop.LearnedCompare(smc.Blueps[len(smc.Blueps)-1]) == 1 {
				prop = smc.Blueps[len(smc.Blueps)-1]
			}
		}
	}

	if i := len(smc.Confs) - 1; i > cur || !regular {

		rst = smc.WriteValue(val, rst)
		setS, err := smc.Confs[i].SetState(&pb.NewState{CurC: uint32(smc.Blueps[i].Len()), Cur: smc.Blueps[i], State: rst, LAState: las})
		cnt++
		if err != nil {
			//Not sure what to do:
			glog.Errorf("C%d: SetState returned error, not sure what to do\n", smc.ID)
			return nil, 0, err
		}
		if i > 0 && glog.V(3) {
			glog.Infof("C%d: Set State in Configuration with length %d\n ", smc.ID, smc.Blueps[i].Len())
		} else if glog.V(6) {
			glog.Infoln("Set state returned.")
		}

		cur = smc.handleOneCur(i, setS.Reply.GetCur())
		smc.handleNext(i, setS.Reply.GetNext())
		if !regular && i+1 < len(smc.Confs) {
			prop = smc.Blueps[len(smc.Blueps)-1]
			goto forconfiguration
		}
	}

	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return rst, cnt, nil
}


func (smc *SmClient) lagree(prop *pb.Blueprint) (dec *pb.Blueprint, cnt int, err error) {
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
			glog.Infof("C%d: LAProp returned.\n", smc.ID)
		}

		cur = smc.handleOneCur(cur, laProp.Reply.GetCur())
		la := laProp.Reply.GetLAState()
		if la != nil && !prop.LearnedEquals(la) {
			if glog.V(3) {
				glog.Infof("C%d: LAProp returned new state, try again.\n", smc.ID)
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

func (smc *SmClient) doread(curin, i int) (st *pb.State, next *pb.Blueprint, cur int, err error) {
	read, errx := smc.Confs[i].AReadS(&pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[i].Len())})
	if errx != nil {
		glog.Errorf("C%d: error from AReadS: ", errx)
		return nil, nil, 0, errx
		//return
	}
	if glog.V(6) {
		glog.Infof("C%d: AReadS returned with replies from \n", smc.ID, read.MachineIDs)
	}
	cur = smc.handleNewCur(curin, read.Reply.GetCur())

	smc.handleNext(i, read.Reply.GetNext())
	
	if len(read.Reply.GetNext()) == 1 {
		// Only used in consreconf
		next = read.Reply.GetNext()[0]
	}

	return read.Reply.GetState(), next, cur, nil
}

func (smc *SmClient) WriteValue(val []byte, st *pb.State) *pb.State {
	if val == nil {
		return st
	}
	return &pb.State{Value: val, Timestamp: st.Timestamp + 1, Writer: smc.ID}
}
