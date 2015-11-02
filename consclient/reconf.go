package consclient

import (
	"errors"
	"time"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (cc *CClient) Reconf(prop *pb.Blueprint) (cnt int, err error) {
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(cc.Blueps[0]) == 1 {
		glog.V(3).Infof("C%d: Proposal is already in place.", cc.ID)
		return 0, nil
	}
	_, cnt, err = cc.reconf(prop, true, nil)
	return
}

func (cc *CClient) reconf(prop *pb.Blueprint, regular bool, val []byte) (rst *pb.State, cnt int, err error) {
	if glog.V(6) {
		glog.Infof("C%d: Starting reconfiguration\n", cc.ID)
	}


	cur := 0
forconfiguration:
	for i := 0; i < len(cc.Confs); i++ {
		if i < cur {
			continue
		}

		var next *pb.Blueprint

		switch prop.Compare(cc.Blueps[i]) {
		case 0, -1:
			//Need to agree on new proposal
			var cs int
			next, cs, cur, err = cc.getconsensus(i, prop)
			if err != nil {
				return nil, 0, err
			}
			cnt += cs
		case 1:
			// No proposal
			var st *pb.State
			st, next, cur, err = cc.doread(cur, i)
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

		if cc.Blueps[i].LearnedCompare(next) == 1 {
			readS, err := cc.Confs[i].CWriteN(&pb.DRead{CurC: uint32(cc.Blueps[i].Len()), Prop: next})
			if glog.V(3) {
				glog.Infof("C%d: CWriteN returned.\n", cc.ID)
			}
			cnt++
			cur = cc.handleOneCur(cur, readS.Reply.GetCur())
			if err != nil && cur <= i {
				glog.Errorf("C%d: error from CReadS: %v\n", cc.ID, err)
				//No Quorum Available. Retry
				return nil, 0, err
			}

			for _, next = range readS.Reply.GetNext() {
				cc.handleNext(i, next)
			}

			if rst.Compare(readS.Reply.GetState()) == 1 {
				rst = readS.Reply.GetState()
			}

		} else if next != nil {
			glog.Errorln("This case should never happen. There might be a bug in the code.")
		}

	}

	if i := len(cc.Confs) - 1; i > cur || !regular {

		rst = cc.WriteValue(val, rst)

		_, err := cc.Confs[i].CSetState(&pb.CNewCur{Cur: cc.Blueps[i], CurC: uint32(cc.Blueps[i].Len()), State: rst})
		if i > 0 && glog.V(3) {
			glog.Infof("C%d: Set state in configuration of size %d.\n", cc.ID, cc.Blueps[i].Len())
		} else if glog.V(6) {
			glog.Infof("Set state returned.")
		}
		
		cnt++
		if err != nil {
			//Not sure what to do:
			glog.Errorf("C%d: SetState returned error, not sure what to do\n", cc.ID)
			return nil, 0, err
		}
		cur = i
	}

	cc.Blueps = cc.Blueps[cur:]
	cc.Confs = cc.Confs[cur:]

	return rst, cnt, nil
}

func (cc *CClient) handleOneCur(cur int, newCur *pb.Blueprint) int {
	if newCur == nil {
		return cur
	}

	if glog.V(3) {
		glog.Infof("C%d: Found new Cur with length %d, current has length %d\n", cc.ID, newCur.Len(), cc.Blueps[cur].Len())
	}
	return cc.findorinsert(cur, newCur)

}

func (cc *CClient) getconsensus(i int, prop *pb.Blueprint) (next *pb.Blueprint, cnt, cur int, err error) {
	ms := 1 * time.Millisecond
	rnd := cc.ID
prepare:
	for {
		//Send Prepare:
		promise, errx := cc.Confs[i].CPrepare(&pb.Prepare{CurC: uint32(cc.Blueps[i].Len()), Rnd: rnd})
		if errx != nil {
			//Should log this for debugging
			glog.Errorf("C%d: Prepare returned error: %v\n", cc.ID, errx)
			return nil, 0, i, errx
		}
		cnt++
		cur = cc.handleOneCur(i, promise.Reply.GetCur())
		if i < cur {
			glog.V(3).Infof("C%d: Prepare returned new current conf.\n", cc.ID)
			return nil, cnt, cur, nil
		}

		rrnd := promise.Reply.Rnd
		switch {
		case promise.Reply.GetDec() != nil:
			next = promise.Reply.GetDec()
			if glog.V(3) {
				glog.Infof("C%d: Promise reported decided value.\n", cc.ID)
			}
			return
		case rrnd <= rnd:
			if promise.Reply.GetVal() != nil {
				next = promise.Reply.Val.Val
				if glog.V(3) {
					glog.Infof("C%d: Re-propose a value.\n", cc.ID)
				}
			} else {
				if glog.V(3) {
					glog.Infof("C%d: Proposing my value.\n", cc.ID)
				}
				if len(prop.Add) == 0 {
					glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
					return nil, cnt, cur, errors.New("Abort before proposing unacceptable configuration.")
				}
				next = prop.Merge(cc.Blueps[i])
			}
		case rrnd > rnd:
			// Increment round, sleep then return to prepare.
			if glog.V(3) {
				glog.Infof("C%d: Conflict, sleeping %v.\n", cc.ID, ms)
			}
			if rrid := rrnd % 256; rrid < cc.ID {
				rnd = rrnd - rrid + cc.ID
			} else {
				rnd = rrnd - rrid + 256 + cc.ID
			}
			time.Sleep(ms)
			ms = 2 * ms
			continue prepare

		}

		learn, errx := cc.Confs[i].CAccept(&pb.Propose{CurC: uint32(cc.Blueps[i].Len()), Val: &pb.CV{rnd, next}})
		if err != nil {
			glog.Errorf("C%d: Accept returned error: %v\n", cc.ID, errx)
			return nil, 0, cur, errx
		}

		cnt++
		cur = cc.handleOneCur(cur, learn.Reply.GetCur())
		if i < cur {
			glog.V(3).Infof("C%d: Accept returned new current conf.\n", cc.ID)
			return
		}

		if learn.Reply.GetDec() == nil && !learn.Reply.Learned {
			if glog.V(3) {
				glog.Infof("C%d: Did not learn, redo prepare.\n", cc.ID)
			}
			rnd += 256
			continue prepare
		}

		if learn.Reply.GetDec() != nil {
			next = learn.Reply.GetDec()
		}

		return
	}
}

func (cc *CClient) doread(curin, i int) (st *pb.State, next *pb.Blueprint, cur int, err error) {
	read, errx := cc.Confs[i].CReadS(&pb.Conf{uint32(cc.Blueps[i].Len()), uint32(cc.Blueps[i].Len())})
	if errx != nil {
		glog.Errorf("C%d: error from CReadS: ", errx)
		return nil, nil, 0, errx
		//return
	}
	if glog.V(6) {
		glog.Infof("C%d: CReadS returned with replies from \n", cc.ID, read.MachineIDs)
	}
	cur = cc.handleNewCur(curin, read.Reply.GetCur())

	var j int
	for j, next = range read.Reply.GetNext() {
		cc.handleNext(i, next)
		if j > 0 {
			glog.Errorln("CReadS returned more than one Next value.")
		}
	}

	return read.Reply.GetState(), next, cur, nil
}

func (cc *CClient) WriteValue(val []byte, st *pb.State) *pb.State {
	if val == nil {
		return st
	}
	return &pb.State{Value: val, Timestamp: st.Timestamp + 1, Writer: cc.ID}
}
