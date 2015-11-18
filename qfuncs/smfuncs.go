package qfuncs

import (
	"github.com/golang/glog"
	pr "github.com/relab/smartMerge/proto"
)

type ConfResponder interface {
	GetCur() *pr.ConfReply
}

func checkConfResponder(cr ConfResponder) bool {
	return checkConfReply(cr.GetCur())
}

func checkConfReply(cr *pr.ConfReply) bool {
	if cr != nil && cr.Abort {
		return true
	} 
	return false
} 

func handleConfResponder(old *pr.ConfReply, cr ConfResponder) *pr.ConfReply {
	return handleConfReply(old, cr.GetCur())
}

func handleConfReply(old *pr.ConfReply, cr *pr.ConfReply) *pr.ConfReply {
	if old == nil {
		return cr
	}
	
	if cr == nil {
		return old
	}
	if old.Cur.LearnedCompare(cr.Cur) == 1 {
		old.Cur = cr.Cur
	}
	
	old.Next = GetBlueprintSlice(old.Next, cr)
	return old
}

var AReadSQF = func(c *pr.Configuration, replies []*pr.ReadReply) (*pr.ReadReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if checkConfResponder(lastrep) {
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
		lastrep.Cur = handleConfResponder(lastrep.Cur, rep) // I think the assignment can be omitted.
	}

	return lastrep, true
}

var AWriteSQF = func(c *pr.Configuration, replies []*pr.ConfReply) (*pr.ConfReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if checkConfReply(lastrep) {
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

	lastrep = new(pr.ConfReply)
	for _, rep := range replies {
		lastrep = handleConfReply(lastrep, rep)
	}

	return lastrep, true
}

var AWriteNQF = func(c *pr.Configuration, replies []*pr.WriteNReply) (*pr.WriteNReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if checkConfResponder(lastrep) {
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
	for _, rep := range replies {
		if lastrep.GetState().Compare(rep.GetState()) == 1 {
			lastrep.State = rep.GetState()
		}
		lastrep.LAState = lastrep.GetLAState().Merge(rep.GetLAState())
		lastrep.Cur = handleConfResponder(lastrep.Cur, rep)
	}

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
	if checkConfResponder(lastrep) {
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
	for _, rep := range replies {
		lastrep.LAState = lastrep.GetLAState().Merge(rep.GetLAState())
		lastrep.Cur = handleConfResponder(lastrep.Cur, rep)
	}

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
	repNext := rep.GetNext()
	if repNext == nil {
		return next
	}
	
	if next == nil {
		next = make([]*pr.Blueprint,0,len(repNext))
	}
	for _, blp := range repNext {
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
