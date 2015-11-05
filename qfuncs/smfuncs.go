package qfuncs

import (
	"github.com/golang/glog"
	pr "github.com/relab/smartMerge/proto"
)

var AReadSQF = func(c *pr.Configuration, replies []*pr.ReadReply) (*pr.ReadReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur().GetCur() != nil && lastrep.GetCur().Abort {
		if glog.V(3) {
			glog.Infoln("ReadS reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	if len(replies) < c.ReadQuorum() {
		if glog.V(7) {
			glog.Infoln("Not enough ReadSReplies yet.")
		}
		return nil, false
	}

	lastrep = new(pr.ReadReply)
	for _, rep := range replies {
		if lastrep.GetState().Compare(rep.GetState()) == 1 {
			lastrep.State = rep.GetState()
		}
		if rep.GetCur() != nil {
			if rep.GetCur().GetCur().Len() > lastrep.GetCur().GetCur().Len() {
				lastrep.Cur = rep.Cur
			}
		}
	}

	next := make([]*pr.Blueprint, 0, 1)
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}

	lastrep.Next = next

	return lastrep, true
}

var AWriteSQF = func(c *pr.Configuration, replies []*pr.WriteSReply) (*pr.WriteSReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur().GetCur() != nil && lastrep.GetCur().Abort {
		if glog.V(3) {
			glog.Infoln("WriteS reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < c.MaxQuorum() {
		if glog.V(7) {
			glog.Infoln("Not enough WriteSReplies yet.")
		}
		return nil, false
	}

	lastrep = new(pr.WriteSReply)
	next := make([]*pr.Blueprint, 0, 1)
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
		if rep.GetCur() != nil {
			if rep.GetCur().GetCur().Len() > lastrep.GetCur().GetCur().Len() {
				lastrep.Cur = rep.Cur
			}
		}
	}

	lastrep.Next = next

	return lastrep, true
}

var AWriteNQF = func(c *pr.Configuration, replies []*pr.WriteNReply) (*pr.WriteNReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		if glog.V(3) {
			glog.Infoln("WriteN reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < c.MaxQuorum() {
		return nil, false
	}

	lastrep = new(pr.WriteNReply)
	for i, rep := range replies {
		if i == len(replies)-1 {
			break
		}
		if lastrep.GetState().Compare(rep.GetState()) == 1 {
			lastrep.State = rep.GetState()
		}
		lastrep.LAState = lastrep.GetLAState().Merge(rep.GetLAState())
	}

	next := make([]*pr.Blueprint, 0, 1)
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}

	lastrep.Next = next

	return lastrep, true
}

var SetCurQF = func(c *pr.Configuration, replies []*pr.NewCurReply) (*pr.NewCurReply, bool) {
	// Return false, if not enough replies yet.
	if len(replies) < c.WriteQuorum() {
		return nil, false
	}

	for _, rep := range replies {
		if rep != nil && !rep.New {
			return rep, true
		}
	}
	return replies[0], true
}

var LAPropQF = func(c *pr.Configuration, replies []*pr.LAReply) (*pr.LAReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		if glog.V(3) {
			glog.Infoln("LAProp reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < c.MaxQuorum() {
		return nil, false
	}

	lastrep = new(pr.LAReply)
	for i, rep := range replies {
		if i == len(replies)-1 {
			break
		}
		lastrep.LAState = lastrep.GetLAState().Merge(rep.GetLAState())
	}

	next := make([]*pr.Blueprint, 0, 1)
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}

	lastrep.Next = next

	return lastrep, true
}

var SetStateQF = func(c *pr.Configuration, replies []*pr.NewStateReply) (*pr.NewStateReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	if len(replies) < c.MaxQuorum() {
		return nil, false
	}

	next := make([]*pr.Blueprint, 0, 1)
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}

	lastrep.Next = next

	return lastrep, true
}

type NextReport interface {
	GetNext() []*pr.Blueprint
}

var GetPromiseQF = func(c *pr.Configuration, replies []*pr.Promise) (*pr.Promise, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < c.ReadQuorum() {
		return nil, false
	}

	lastrep = new(pr.Promise)
	for _, rep := range replies {
		if rep == nil {
			continue
		}

		if rep.GetDec() != nil {
			return rep, true
		}

		if rep.Rnd > lastrep.Rnd {
			lastrep.Rnd = rep.Rnd
		}
		if rep.Val == nil {
			continue
		}
		if lastrep.Val == nil || rep.Val.Rnd > lastrep.Val.Rnd {
			lastrep.Val = rep.Val
		}
	}

	return lastrep, true
}

var AcceptQF = func(c *pr.Configuration, replies []*pr.Learn) (*pr.Learn, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < c.MaxQuorum() {
		return nil, false
	}

	lastrep = new(pr.Learn)
	lastrep.Learned = true
	for _, rep := range replies {
		if rep == nil || !rep.Learned {
			lastrep.Learned = false
		}

		if rep.GetDec() != nil {
			return rep, true
		}
	}

	return lastrep, true

}

func GetBlueprintSlice(next []*pr.Blueprint, rep NextReport) []*pr.Blueprint {
	for _, blp := range rep.GetNext() {
		next = addLearned(next, blp)
	}

	return next
}

func addLearned(bls []*pr.Blueprint, bp *pr.Blueprint) []*pr.Blueprint {
	place := 0

findplacefor:
	for _, blpr := range bls {
		switch blpr.LearnedCompare(bp) {
		case 0:
			//New blueprint already present
			return bls
		case -1:
			break findplacefor
		default:
			place += 1
			continue
		}
	}

	bls = append(bls, nil)

	for i := len(bls) - 1; i > place; i-- {
		bls[i] = bls[i-1]
	}
	bls[place] = bp

	return bls
}

type LAStateReport interface {
	GetLAState() *pr.Blueprint
}

func MergeLAState(las *pr.Blueprint, rep LAStateReport) *pr.Blueprint {
	lap := rep.GetLAState()
	if lap == nil {
		return las
	}
	if las == nil {
		return lap
	}
	return las.Merge(lap)
}

type CurReport interface {
	GetCur() *pr.Blueprint
}
