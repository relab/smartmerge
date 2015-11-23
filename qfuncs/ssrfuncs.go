package qfuncs

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

var SpSnOneQF = func(c *pb.Configuration, replies []*pb.SWriteNReply) (*pb.SWriteNReply, bool) {

	lastrep := replies[len(replies)-1]
	if lastrep.GetCur != nil {
		if glog.V(4) {
			glog.Infoln("SWriteNReply reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	if len(replies) < c.MaxQuorum() {
		if glog.V(7) {
			glog.Infoln("Not enough SWriteNReplies yet.")
		}
		return nil, false
	}

	var next []*pb.Blueprint
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}

	return &pb.SWriteNReply{Next: next}, true
}

var SCommitQF = func(c *pb.Configuration, replies []*pb.CommitReply) (*pb.CommitReply, bool) {

	lastrep := replies[len(replies)-1]
	if lastrep.GetCur != nil {
		if glog.V(4) {
			glog.Infoln("SCommitReply reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	if len(replies) < c.MaxQuorum() {
		if glog.V(7) {
			glog.Infoln("Not enough CommitReplies yet.")
		}
		return nil, false
	}

	lastrep = new(pb.CommitReply)
	for _, rep := range replies {
		if lastrep.Committed == nil {
			lastrep.Committed = rep.Committed
		}
		lastrep.Collected = lastrep.Collected.Merge(rep.Collected)
	}

	return lastrep, true

}

var SReadSQF = func(c *pb.Configuration, replies []*pb.SReadReply) (*pb.SReadReply, bool) {

	lastrep := replies[len(replies)-1]
	if lastrep.GetCur != nil {
		if glog.V(4) {
			glog.Infoln("SReadReply reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	if len(replies) < c.ReadQuorum() {
		if glog.V(7) {
			glog.Infoln("Not enough SReadReplies yet.")
		}
		return nil, false
	}

	rst := new(pb.SReadReply)
	for _, rep := range replies {
		if rst.State.Compare(rep.State) == 1 {
			rst.State = rep.State
		}
	}

	return rst, true

}

var SSetStateQF = func(c *pb.Configuration, replies []*pb.SStateReply) (*pb.SStateReply, bool) {
	//Oups here we don't abort when a new cur is reported, since this is not always processed.

	// Return false, if not enough replies yet.
	if len(replies) < c.MaxQuorum() {
		if glog.V(7) {
			glog.Infoln("Not enough SReadReplies yet.")
		}
		return nil, false
	}

	therep := new(pb.SStateReply)
	for _, rep := range replies {
		if rep.HasNext {
			therep.HasNext = true
		}

		if therep.Cur.LearnedCompare(rep.Cur) == 1 {
			therep.Cur = rep.Cur
		}
	}

	return therep, true

}
