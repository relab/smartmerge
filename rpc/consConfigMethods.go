package rpc

import (
	"golang.org/x/net/context"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	//"github.com/relab/smartMerge/regserver"
)

func (c *Configuration) CReadS(thisBP *lat.Blueprint, curC uint32, prop *lat.Blueprint) (s *pb.State, next *lat.Blueprint, newCur *lat.Blueprint, err error) {
	mprop := prop.ToMsg()
	replies, newCur, err := c.mgr.CReadS(c.id, thisBP, &pb.DRead{CurC: curC, Prop: mprop}, context.Background())
	if err != nil || newCur != nil {
		return
	}

	for _, rep := range replies {
		if s.Compare(rep.GetState()) == 1 {
			s = rep.GetState()
		}
		next = CheckNext(next, rep)
		newCur = CompareCur(newCur, rep)
	}
	return
}


func (c *Configuration) CWriteS(s *pb.State, thisBP *lat.Blueprint, curC uint32) (next *lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.CWriteS(c.id, thisBP, context.Background(), &pb.AdvWriteS{s, curC})
	if err != nil || newCur != nil {
		return
	}

	for _, rep := range replies {
		next = CheckNext(next, rep)
		newCur = CompareCur(newCur, rep)
	}
	return

}

func (c *Configuration) CPrepare(thisBP *lat.Blueprint, newrnd uint32) (rnd uint32, decided bool, backup bool, next *lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.CPrepare(c.id, thisBP, context.Background(), newrnd)
	if err != nil || newCur != nil {
		return

	}

	tmpval := new(pb.CV)
	for _, rep := range replies {
		rnd = CheckRnd(rnd, rep)
		next = CheckDec(next, rep)
		newCur = CompareCur(newCur, rep)
		tmpval = CheckVal(tmpval, rep)
	}
	if rnd > tmpval.Rnd {
		backup = true
	} else {
		rnd = tmpval.Rnd
	}	
	if next != nil {
		decided = true
	} else {
		next = lat.GetBlueprint(tmpval.Val)
		
	}
	
	return 
}

func (c *Configuration) CAccept(thisBP *lat.Blueprint, rnd uint32, val *lat.Blueprint) (dec *lat.Blueprint, learned bool, newCur *lat.Blueprint, err error) {
	pbval := pb.CV{rnd, val.ToMsg()}
	replies, newCur, err := c.mgr.CAccept(c.id, thisBP, context.Background(), &pb.Propose{c.id, &pbval})
	if err != nil || newCur != nil {
		return
	}

	learned = true
	for _, rep := range replies {
		newCur = CompareCur(newCur, rep)
		dec = CheckDec(dec, rep)
		if rep != nil && !rep.Learned {
			learned = false
		}
	}
	if dec == nil && learned {
		dec = val
	}
	
	return
}

func (c *Configuration) CSetState(cur *lat.Blueprint, st *pb.State) error {
	msgCur := cur.ToMsg()

	_, err := c.mgr.CSetState(c.id, context.Background(), &pb.CNewCur{Cur: msgCur, CurC: c.id, State: st})
	if err != nil {
		return err
	}
	return nil
}

func CheckNext(next *lat.Blueprint, rep NextReport) *lat.Blueprint {
	if next != nil {
		return next
	}
	nextsl := rep.GetNext()
	if len(nextsl)> 0 && nextsl[0]!= nil {
		return lat.GetBlueprint(nextsl[0])
	}
	return next
}

type DecReport interface {
	GetDec() *pb.Blueprint
}

func CheckDec(dec *lat.Blueprint, rep DecReport) *lat.Blueprint {
	if dec != nil {
		return dec
	}
	pbdec := rep.GetDec()
	return lat.GetBlueprint(pbdec)
}

func CheckRnd(rnd uint32, rep *pb.Promise) uint32 {
	if rep == nil {
		return rnd
	}
	if rep.Rnd > rnd {
		return rep.Rnd
	}
	return rnd
}

func CheckVal(val *pb.CV, rep *pb.Promise) *pb.CV {
	if rep == nil {
		return val
	}
	if rep.Val == nil {
		return val
	}
	if val == nil {
		return rep.Val
	}
	if val.Rnd < rep.Val.Rnd {
		return rep.Val
	}
	return val
}