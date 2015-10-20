package qfuncs

import (

	pr "github.com/relab/smartMerge/proto"

)

var CReadSQF = AReadSQF
var CWriteSQF = AWriteSQF
var CSetStateQF = SetCurQF

var CPrepareQF = func(c *pr.Configuration, replies []*pr.Promise) (*pr.Promise, bool){
	
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
	for _,rep := range replies {
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
		if rep.Val.Rnd > lastrep.Val.Rnd {
			lastrep.Val = rep.Val
		}
	}
	
	return lastrep, true
}

var CAcceptQF = func(c *pr.Configuration, replies []*pr.Learn) (*pr.Learn, bool) {

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
	for _,rep := range replies {
		if rep == nil || !rep.Learned {
			lastrep.Learned = false
		}
		
		if rep.GetDec() != nil {
			return rep, true
		}
	}
	
	return lastrep, true
	
}