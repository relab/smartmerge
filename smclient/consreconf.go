package smclient

import (
	"errors"
	"time"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) consreconf(prop *pb.Blueprint, regular bool, val []byte) (rst *pb.State, cnt int, err error) {
	if glog.V(6) {
		glog.Infof("C%d: Starting reconfiguration\n", smc.ID)
	}

	doconsensus := true
	cur := 0
forconfiguration:
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		var next *pb.Blueprint

		switch prop.Compare(smc.Blueps[i]) {
		case 0, -1:
			if doconsensus {
				//Need to agree on new proposal
				var cs int
				next, cs, cur, err = smc.getconsensus(i, prop)
				if err != nil {
					return nil, 0, err
				}
				cnt += cs
			} else {
				next = prop
			}
		case 1:
			// No proposal
			var st *pb.State
			st, cur, err = smc.doread(cur, i)
			if err != nil {
				return nil, 0, err
			}
			if i+1 < len(smc.Blueps) {
				next = smc.Blueps[i+1]
			}
			cnt++
			if rst.Compare(st) == 1 {
				rst = st
			}
		}
		if i < cur {
			continue forconfiguration
		}

		if smc.Blueps[i].LearnedCompare(next) == 1 {
			readS, err := smc.Confs[i].AWriteN(&pb.WriteN{CurC: uint32(smc.Blueps[i].Len()), Next: next})
			if glog.V(3) {
				glog.Infof("C%d: CWriteN returned.\n", smc.ID)
			}
			cnt++
			if err != nil && cur <= i {
				glog.Errorf("C%d: error from CReadS: %v\n", smc.ID, err)
				//No Quorum Available. Retry
				return nil, 0, err
			}

			cur = smc.handleNewCur(cur, readS.Reply.GetCur(), true)

			if rst.Compare(readS.Reply.GetState()) == 1 {
				rst = readS.Reply.GetState()
			}

		} else if next != nil {
			glog.Errorln("This case should never happen. There might be a bug in the code.")
		}

	}

	if i := len(smc.Confs) - 1; i > cur || !regular {

		rst = smc.WriteValue(val, rst)

		setS, err := smc.Confs[i].SetState(&pb.NewState{Cur: smc.Blueps[i], CurC: uint32(smc.Blueps[i].Len()), State: rst})
		if i > 0 && glog.V(3) {
			glog.Infof("C%d: Set state in configuration of size %d.\n", smc.ID, smc.Blueps[i].Len())
		} else if glog.V(6) {
			glog.Infof("Set state returned.")
		}

		cnt++
		if err != nil {
			//Not sure what to do:
			glog.Errorf("C%d: SetState returned error, not sure what to do\n", smc.ID)
			return nil, 0, err
		}
		cur = smc.handleOneCur(i, setS.Reply.GetCur(), true)
		smc.handleNext(i, setS.Reply.GetNext(), true)

		if !regular && i < len(smc.Confs)-1 {
			prop = smc.Blueps[len(smc.Blueps)-1]
			doconsensus = false
			goto forconfiguration
		}
	}

	smc.Blueps = smc.Blueps[cur:]
	smc.Confs = smc.Confs[cur:]

	return rst, cnt, nil
}

func (smc *SmClient) getconsensus(i int, prop *pb.Blueprint) (next *pb.Blueprint, cnt, cur int, err error) {
	ms := 1 * time.Millisecond
	rnd := smc.ID
prepare:
	for {
		//Send Prepare:
		promise, errx := smc.Confs[i].GetPromise(&pb.Prepare{CurC: uint32(smc.Blueps[i].Len()), Rnd: rnd})
		if errx != nil {
			//Should log this for debugging
			glog.Errorf("C%d: Prepare returned error: %v\n", smc.ID, errx)
			return nil, 0, i, errx
		}
		cnt++
		cur = smc.handleOneCur(i, promise.Reply.GetCur(), true)
		if i < cur {
			glog.V(3).Infof("C%d: Prepare returned new current conf.\n", smc.ID)
			return nil, cnt, cur, nil
		}

		rrnd := promise.Reply.Rnd
		switch {
		case promise.Reply.GetDec() != nil:
			next = promise.Reply.GetDec()
			if glog.V(3) {
				glog.Infof("C%d: Promise reported decided value.\n", smc.ID)
			}
			return
		case rrnd <= rnd:
			if promise.Reply.GetVal() != nil {
				next = promise.Reply.Val.Val
				if glog.V(3) {
					glog.Infof("C%d: Re-propose a value.\n", smc.ID)
				}
			} else {
				if glog.V(3) {
					glog.Infof("C%d: Proposing my value.\n", smc.ID)
				}
				if len(prop.Ids()) == 0 {
					glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
					return nil, cnt, cur, errors.New("Abort before proposing unacceptable configuration.")
				}
				next = prop.Merge(smc.Blueps[i])
			}
		case rrnd > rnd:
			// Increment round, sleep then return to prepare.
			if glog.V(3) {
				glog.Infof("C%d: Conflict, sleeping %v.\n", smc.ID, ms)
			}
			if rrid := rrnd % 256; rrid < smc.ID {
				rnd = rrnd - rrid + smc.ID
			} else {
				rnd = rrnd - rrid + 256 + smc.ID
			}
			time.Sleep(ms)
			ms = 2 * ms
			continue prepare

		}

		learn, errx := smc.Confs[i].Accept(&pb.Propose{CurC: uint32(smc.Blueps[i].Len()), Val: &pb.CV{rnd, next}})
		if err != nil {
			glog.Errorf("C%d: Accept returned error: %v\n", smc.ID, errx)
			return nil, 0, cur, errx
		}

		cnt++
		cur = smc.handleOneCur(cur, learn.Reply.GetCur(), true)
		if i < cur {
			glog.V(3).Infof("C%d: Accept returned new current conf.\n", smc.ID)
			return
		}

		if learn.Reply.GetDec() == nil && !learn.Reply.Learned {
			if glog.V(3) {
				glog.Infof("C%d: Did not learn, redo prepare.\n", smc.ID)
			}
			rnd += 256
			continue prepare
		}

		if learn.Reply.GetDec() != nil {
			next = learn.Reply.GetDec()
		}

		glog.V(4).Infof("C%d: Did Learn a value.", smc.ID)
		return
	}
}
