package smclient

import (
	"errors"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmOptClient) Reconf(prop *pb.Blueprint) (cnt int, err error) {
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(smc.Blueps[0]) == 1 {
		glog.V(3).Infof("C%d: Proposal is already in place.", smc.ID)
		return 0, nil
	}
	
	if smc.doCons {
		_, cnt, err = smc.consreconf(prop, true, nil)
	} else {
		_, cnt, err = smc.optreconf(prop, true, nil)
	}
	return
}

func (smc *SmOptClient) optreconf(prop *pb.Blueprint, regular bool, val []byte) (rst *pb.State, cnt int, err error) {
	if glog.V(6) {
		glog.Infof("C%d: Starting reconf\n", smc.ID)
	}

	if prop.Compare(smc.Blueps[0]) != 1 {
		// A new blueprint was proposed. Need to solve Lattice Agreement:
		prop, cnt, err = smc.lagree(prop)
		if err != nil {
			return nil, 0, err
		}
		if len(prop.Ids()) < MinSize  {
			glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
			return nil, cnt, errors.New("Abort before moving to unacceptable configuration.")
		}
	}

	cur := 0
	las := new(pb.Blueprint)
	var rid []uint32

forconfiguration:
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		// If we are in the current configuration, do a read, to check for next configurations. No need to recontact.
		if prop.LearnedCompare(smc.Blueps[i]) != -1 {
			if len(smc.Blueps) > i+1 {
				prop = smc.Blueps[len(smc.Blueps)-1]
				rid = nil	// Empty rid on new Write Value.
			} else if cur == i || !regular {
				// If we are in the current configuration, do a read, to check for next configurations. No need to recontact.
				var st *pb.State
				var c int
				st, _, cur, c, err = smc.doread(cur, i, rid)
				if err != nil {
					return nil, 0, err
				}
				cnt += c
				if rst.Compare(st) == 1 {
					rst = st
				}
				
				if i < cur {
					continue forconfiguration
				}
				
				prop = smc.Blueps[len(smc.Blueps)-1]
				rid = nil	// Empty rid on new Write Value.
			}
		}

		if prop.LearnedCompare(smc.Blueps[i]) == -1 {
			// There exists a proposal => do WriteN
			
			cnf := smc.getWriteC(i, rid)
		
			writeN := new(pb.AWriteNReply)
		
			for j := 0; cnf != nil ; j++ {
				writeN, err = cnf.AWriteN(&pb.WriteN{uint32(smc.Blueps[i].Len()), prop})
				cnt++
			
				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedWriteN: %v\n",smc.ID, err)
					// Try again with full configuration.
					cnf = smc.getFullC(i)
				}
			
				if err != nil && j == Retry{ 
					glog.Errorf("C%d: error %v from WriteN after %d retries: ", smc.ID, err, Retry)
					return nil, 0, err
				}
			
				if err == nil {
					break
				}
			}

			cur = smc.handleOneCur(cur, writeN.Reply.GetCur(), false)
			smc.handleNext(i, writeN.Reply.GetNext(), true)
			las = las.Merge(writeN.Reply.GetLAState())
			if rst.Compare(writeN.Reply.GetState()) == 1 {
				rst = writeN.Reply.GetState()
			}
			
			if write.Reply.GetCur() == nil || !write.Reply.Cur.Abort {
				rid = pb.Union(rid, write.MachineIDs)
			}
		}

		if i := len(smc.Confs) - 1; i > cur || !regular {
    	
			rst = smc.WriteValue(val, rst)
			
			cnf := smc.getWriteC(i, nil)
		
			setS := new(pb.SetStateReply)
		
			for j := 0; cnf != nil ; j++ {
				setS, err = cnf.SetState(&pb.NewState{CurC: uint32(smc.Blueps[i].Len()), Cur: smc.Blueps[i], State: rst, LAState: las})
				cnt++
			
				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedSetState: %v\n",smc.ID, err)
					// Try again with full configuration.
					cnf = smc.getFullC(i)
				}
			
				if err != nil && j == Retry{ 
					glog.Errorf("C%d: error %v from SetState after %d retries: ", smc.ID, err, Retry)
					return nil, 0, err
				}
			
				if err == nil {
					break
				}
			}
			
			if i > 0 && glog.V(3) {
				glog.Infof("C%d: Set State in Configuration with length %d\n ", smc.ID, smc.Blueps[i].Len())
			} else if glog.V(6) {
				glog.Infoln("Set state returned.")
			}
    	
			cur = smc.handleOneCur(i, setS.Reply.GetCur(), false)
			smc.handleNext(i, setS.Reply.GetNext(), true)
			
			if SetS.Reply.GetCur() == nil {
				rid = pb.Union(rid, setS.MachineIDs)
			}
			
		}
	}
	

	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return rst, cnt, nil
}


func (smc *SmOptClient) lagree(prop *pb.Blueprint) (dec *pb.Blueprint, cnt int, err error) {
	cur := 0
	var rid []uint32
	prop = prop.Merge(smc.Blueps[0])
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		cnf := smc.getWriteC(i, rid)
		
		laProp := new(pb.LAPropReply)
		var err error
		
		for j := 0; cnf != nil ; j++ {
			laProp, err = cnf.LAProp(&pb.LAProposal{uint32(smc.Blueps[i].Len()), prop})
			cnt++
			
			if err != nil && j == 0 {
				glog.Errorf("C%d: error from OptimizedLAProp: %v\n",smc.ID, err)
				// Try again with full configuration.
				cnf = smc.getFullC(i)
			}
			
			if err != nil && j == Retry{ 
				glog.Errorf("C%d: error %v from LAProp after %d retries: ", smc.ID, err, Retry)
				return nil, 0, err
			}
			
			if err == nil {
				break
			}
		}

		if glog.V(4) {
			glog.Infof("C%d: LAProp returned.\n", smc.ID)
		}

		cur = smc.handleOneCur(cur, laProp.Reply.GetCur(), false)
		la := laProp.Reply.GetLAState()
		if la != nil && !prop.LearnedEquals(la) {
			if glog.V(3) {
				glog.Infof("C%d: LAProp returned new state, try again.\n", smc.ID)
			}
			prop = la
			i--
			rid = nil 
			continue
		}

		smc.handleNext(i, laProp.Reply.GetNext(), false)
		
		if len(smc.Blueps) > i+1 && laProp.Reply.GetCur() == nil {
			rid = pb.Union(rid, laProp.MachineIDs)
		}
	}

	smc.setNewCur(cur)
	return prop, cnt, nil
}


func (smc *SmOptClient) doread(curin, i int, rid []uint32) (st *pb.State, next *pb.Blueprint, cur, cnt int, err error) {
	cnf := smc.getReadC(i, rid)
	
	read := new(pb.AReadSReply)
	
	for j := 0; cnf != nil ; j++ {
		read, err = cnf.AReadS(&pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[i].Len())})
		cnt++
		
		if err != nil && j == 0 {
			glog.Errorf("C%d: error from OptimizedReads: %v\n",smc.ID, err)
			// Try again with full configuration.
			cnf = smc.getFullC(i)
		}
		
		if err != nil && j == Retry{ 
			glog.Errorf("C%d: error %v from ReadS after %d retries: ", smc.ID, err, Retry)
			return nil, nil, 0, 0, err
		}
		
		if err == nil {
			break
		}
	}
	
	if glog.V(6) {
		glog.Infof("C%d: AReadS returned with replies from \n", smc.ID, read.MachineIDs)
	}
	cur = smc.handleNewCur(curin, read.Reply.GetCur(), false)

	smc.handleNext(i, read.Reply.GetNext(), false)
	
	if len(read.Reply.GetNext()) == 1 {
		// Only used in consreconf
		next = read.Reply.GetNext()[0]
	}

	return read.Reply.GetState(), next, cur, nil
}