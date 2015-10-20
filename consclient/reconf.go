package consclient

import (
	"errors"
	"fmt"
	"time"

	pb "github.com/relab/smartMerge/proto"
)

func (cc *CClient) Reconf(prop *pb.Blueprint) (cnt int, err error) {
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(cc.Blueps[0]) == 1 {
		return cnt, nil
	}

	if len(prop.Add) == 0 {
		return cnt, errors.New("Abort before proposing unacceptable configuration.")
	}

	cur := 0
	
	var (
		rrnd uint32
		next *pb.Blueprint
		promise *pb.CPrepareReply
		learn *pb.CAcceptReply
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
		promise, err = cc.Confs[i].CPrepare(&pb.Prepare{CurC: uint32(cc.Blueps[i].Len()),Rnd: rnd})
		cnt++
		cur = cc.handleNewCur(cur, promise.Reply.GetCur())
		if i < cur {
			continue
		}
		
		if err != nil {
			//Should log this for debugging
			fmt.Println("Prepare returned error: ",err)
			panic("Error from CPrepare")
		}

		rrnd = promise.Reply.Rnd
		switch {
		case promise.Reply.GetDec() != nil:
			next = promise.Reply.GetDec()
			goto decide
		case rrnd <= rnd:
			if promise.Reply.GetVal() != nil {
				next = promise.Reply.Val.Val
			} else {
				next = prop.Merge(cc.Blueps[i])
			}
			
		case rrnd > rnd: 
			if rrid := rrnd%256; rrid < cc.ID {
				rnd = rrnd-rrid+cc.ID
			} else {
				rnd = rrnd-rrid+256+cc.ID
			}
			time.Sleep(ms)
			ms = 2*ms
			goto prepare
		}
		
	
		learn, err = cc.Confs[i].CAccept(&pb.Propose{CurC: uint32(cc.Blueps[i].Len()),Val: &pb.CV{rnd, next}})
		cnt++
		cur = cc.handleNewCur(cur, learn.Reply.GetCur())
		if i < cur {
			continue
		}
		
		if err != nil {
			//Should log this for debugging
			fmt.Println("Accept returned error: ",err)
			panic("Error from CAccept")
		}

		if learn.Reply.GetDec() == nil && !learn.Reply.Learned {
			rnd += 256
			goto prepare
		}
		
		if learn.Reply.GetDec() != nil {
			next = learn.Reply.GetDec()
		}
		
		decide:
		readS, err := cc.Confs[i].CReadS(&pb.DRead{CurC: uint32(cc.Blueps[i].Len()),Prop: next})
		cnt++
		cur = cc.handleNewCur(cur, readS.Reply.GetCur())
		if err != nil && cur <= i {
			fmt.Println("error from CReadS: ", err)
			//No Quorum Available. Retry
			panic("Cread returned error")
		}
		
		for _, next = range readS.Reply.GetNext() {
			cc.handleNext(i, next)
		}
		
		
		if rst.Compare(readS.Reply.GetState()) == 1 {
			rst = readS.Reply.GetState()
		}
	}

	if i := len(cc.Confs) - 1; i > cur {
		_, err := cc.Confs[i].CSetState(&pb.CNewCur{Cur:cc.Blueps[i], CurC: uint32(cc.Blueps[i].Len()),State: rst})
		cnt++
		if err != nil {
			//Not sure what to do:
			fmt.Println("SetState returned error, not sure what to do")
			panic("Error from SetState")
		}
		cur = i
	}
	
	cc.Blueps = cc.Blueps[cur:]
	cc.Confs = cc.Confs[cur:]
	
	return cnt, nil
}
