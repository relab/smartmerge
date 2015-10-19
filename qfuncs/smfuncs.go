package qfuncs

import (

	pr "github.com/relab/smartMerge/proto"

)

var AReadSQF = func(c *pr.Configuration, replies []*pr.AdvReadReply) (*pr.AdvReadReply, bool){
	
	// Stop RPC if new current configuration reported. 
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		return lastrep, true
	}
	
	// Return false, if not enough replies yet.
	if len(replies) < c.ReadQuorum() {
		return nil, false
	}
	
	lastrep = new(pr.AdvReadReply)
	for _,rep := range replies {
		if lastrep.GetState().Compare(rep.GetState()) == 1 {
			lastrep.State = rep.GetState()
		}
	}
	
	next := make([]*pr.Blueprint,0,1)
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}
	
	lastrep.Next = next
	
	return lastrep, true	
}


var AWriteSQF = func(c *pr.Configuration, replies []*pr.AdvWriteSReply) (*pr.AdvWriteSReply, bool) {
	
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
	
	lastrep = new(pr.AdvWriteSReply)
	next := make([]*pr.Blueprint,0,1)
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}
	
	lastrep.Next = next
	
	return lastrep, true
}

var AWriteNQF = func(c *pr.Configuration, replies []*pr.AdvWriteNReply) (*pr.AdvWriteNReply, bool) {
	
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
	
	lastrep = new(pr.AdvWriteNReply)
	for i, rep := range replies {
		if i == len(replies)-1 {
			break
		}
		if lastrep.GetState().Compare(rep.GetState()) == 1 {
			lastrep.State = rep.GetState()
		}
		lastrep.LAState = lastrep.GetLAState().Merge(rep.GetLAState())
	}
	
	next := make([]*pr.Blueprint,0,1)
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
	
	next := make([]*pr.Blueprint,0,1)
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
	if len(replies) < c.WriteQuorum() {
		return nil, false
	}
	
	return nil, true
}


type NextReport interface {
	GetNext() []*pr.Blueprint
}

func GetBlueprintSlice(next []*pr.Blueprint, rep NextReport) []*pr.Blueprint {
	for _, blp := range rep.GetNext() {
		next = addLearned(next, blp)
	}

	return next
}

func addLearned(bls []*pr.Blueprint, bp *pr.Blueprint) []*pr.Blueprint {
	place := -1
	
	findplacefor:
	for _, blpr := range bls {
		place += 1
		switch blpr.LearnedCompare(bp) {
		case 0:
			//New blueprint already present
			return bls
		case -1:
			break findplacefor
		default:
			continue
		}
	}
	
	bls = append(bls, nil)
	
	for i := len(bls); i >= place; i-- {
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
