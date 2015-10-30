package consclient

import (
	"errors"
	"time"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (cc *CClient) Reconf(prop *pb.Blueprint) (cnt int, err error) {
	if glog.V(2) {
		glog.Infoln("Starting reconfiguration")
	}
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(cc.Blueps[0]) == 1 {
		glog.V(3).Infoln("Proposal is already in place.")
		return cnt, nil
	}

	if len(prop.Add) == 0 {
		glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
		return cnt, errors.New("Abort before proposing unacceptable configuration.")
	}

	cur := 0

	var (
		rrnd    uint32
		next    *pb.Blueprint
		promise *pb.CPrepareReply
		learn   *pb.CAcceptReply
	)
	rst := new(pb.State)
	rnd := cc.ID
	for i := 0; i < len(cc.Confs); i++ {
		if i < cur {
			continue
		}

		ms := 1 * time.Millisecond
		if prop.Compare(cc.Blueps[i]) == 1 {
			//No new proposal, that we need to agree on.
			next = nil
			goto decide
		}

	prepare:
		promise, err = cc.Confs[i].CPrepare(&pb.Prepare{CurC: uint32(cc.Blueps[i].Len()), Rnd: rnd})
		if glog.V(3) {
			glog.Infoln("Prepare returned")
		}
		cnt++
		cur = cc.handleOneCur(cur, promise.Reply.GetCur())
		if i < cur {
			continue
		}

		if err != nil {
			//Should log this for debugging
			glog.Errorln("Prepare returned error: ", err)
			return 0, err
		}

		rrnd = promise.Reply.Rnd
		switch {
		case promise.Reply.GetDec() != nil:
			next = promise.Reply.GetDec()
			if glog.V(3) {
				glog.Infoln("Promise reported decided value.")
			}
			goto decide
		case rrnd <= rnd:
			if promise.Reply.GetVal() != nil {
				next = promise.Reply.Val.Val
				if glog.V(3) {
					glog.Infoln("Re-propose a value.")
				}
			} else {
				next = prop.Merge(cc.Blueps[i])
				if glog.V(3) {
					glog.Infoln("Proposing my value.")
				}
			}

		case rrnd > rnd:
			if glog.V(3) {
				glog.Infoln("Conflict, sleeping %d ms.", ms)
			}
			if rrid := rrnd % 256; rrid < cc.ID {
				rnd = rrnd - rrid + cc.ID
			} else {
				rnd = rrnd - rrid + 256 + cc.ID
			}
			time.Sleep(ms)
			ms = 2 * ms
			goto prepare
		}

		learn, err = cc.Confs[i].CAccept(&pb.Propose{CurC: uint32(cc.Blueps[i].Len()), Val: &pb.CV{rnd, next}})
		if glog.V(3) {
			glog.Infoln("Accept returned.")
		}
		cnt++
		cur = cc.handleOneCur(cur, learn.Reply.GetCur())
		if i < cur {
			continue
		}

		if err != nil {
			//Should log this for debugging
			glog.Errorln("Accept returned error: ", err)
			return 0, err
		}

		if learn.Reply.GetDec() == nil && !learn.Reply.Learned {
			if glog.V(3) {
				glog.Infoln("Did not learn, redo prepare.")
			}
			rnd += 256
			goto prepare
		}

		if learn.Reply.GetDec() != nil {
			next = learn.Reply.GetDec()
		}

	decide:
		readS, err := cc.Confs[i].CWriteN(&pb.DRead{CurC: uint32(cc.Blueps[i].Len()), Prop: next})
		if glog.V(3) {
			glog.Infoln("CWriteN returned.")
		}
		cnt++
		cur = cc.handleOneCur(cur, readS.Reply.GetCur())
		if err != nil && cur <= i {
			glog.Errorln("error from CReadS: ", err)
			//No Quorum Available. Retry
			return 0, err
		}

		for _, next = range readS.Reply.GetNext() {
			cc.handleNext(i, next)
		}

		if rst.Compare(readS.Reply.GetState()) == 1 {
			rst = readS.Reply.GetState()
		}
	}

	if i := len(cc.Confs) - 1; i > cur {
		_, err := cc.Confs[i].CSetState(&pb.CNewCur{Cur: cc.Blueps[i], CurC: uint32(cc.Blueps[i].Len()), State: rst})
		if glog.V(3) {
			glog.Infof("Set state in configuration of size %d.\n", cc.Blueps[i].Len())
		}
		cnt++
		if err != nil {
			//Not sure what to do:
			glog.Errorln("SetState returned error, not sure what to do")
			return 0, err
		}
		cur = i
	}

	cc.Blueps = cc.Blueps[cur:]
	cc.Confs = cc.Confs[cur:]

	return cnt, nil
}

func (cc *CClient) handleOneCur(cur int, newCur *pb.Blueprint) int {
	if newCur == nil {
		return cur
	}
	
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Len(), cc.Blueps[cur].Len())
	}
	return cc.findorinsert(cur, newCur)
	
}
