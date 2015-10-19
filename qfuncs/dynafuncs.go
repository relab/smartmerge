package qfuncs

import (

	pr "github.com/relab/smartMerge/proto"

)

var DReadSQF = func(c *pr.Configuration, replies []*pr.AdvReadReply) (*pr.AdvReadReply, bool){
	
	// Stop RPC if new current configuration reported. 
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		return lastrep, true
	}
	
	// Return false, if not enough replies yet.
	if len(replies) < c.MaxQuorum() {
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

var DWriteSQF = AWriteSQF

var DWriteNSetQF = func(c *pr.Configuration, replies []*pr.DWriteNReply) (*pr.DWriteNReply, bool) {
	
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

var GetOneNQF = func(c *pr.Configuration, replies []*pr.GetOneReply) (*pr.GetOneReply, bool) {
	return replies[0], true
}

var DSetCurQF = SetCurQF
