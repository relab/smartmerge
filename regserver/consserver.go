package regserver

import (
	"errors"

	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
	"github.com/golang/glog"
)

type ConsServer struct {
	*RegServer
}



func NewConsServer() *ConsServer {
	return &ConsServer{
		NewRegServer(),
	}
}

func NewConsServerWithCur(cur *pb.Blueprint, curc uint32) *ConsServer {
	return &ConsServer{
		NewRegServerWithCur(cur, curc),
	}
}

func (cs *ConsServer) AReadS(ctx context.Context, rr *pb.Conf) (*pb.ReadReply, error) {
	cs.RLock()
	defer cs.RUnlock()
	glog.V(5).Infoln("Handling ReadS")

	if rr.This < cs.CurC {
		// The client is in an outdated configuration.
		return &pb.ReadReply{State: nil, Cur: &pb.ConfReply{cs.Cur, true}, Next: nil}, nil
	}
	
	var next []*pb.Blueprint
	if cs.NextMap[rr.This] != nil {
		next = []*pb.Blueprint{cs.NextMap[rr.This]}
	}
	
	if rr.Cur < cs.CurC {
		return &pb.ReadReply{State: cs.RState, Cur: &pb.ConfReply{cs.Cur, false}, Next: next}, nil
	}

	return &pb.ReadReply{State: cs.RState, Next: next}, nil
}

func (cs *ConsServer) AWriteS(ctx context.Context, wr *pb.WriteS) (*pb.WriteSReply, error) {
	cs.Lock()
	defer cs.Unlock()
	glog.V(5).Infoln("Handling WriteS")
	if cs.RState.Compare(wr.State) == 1 {
		cs.RState = wr.State
	}

	if wr.Conf.This < cs.CurC {
		// The client is in an outdated configuration.
		return &pb.WriteSReply{Cur: &pb.ConfReply{cs.Cur, true}}, nil
	}
	
	var next []*pb.Blueprint
	if cs.NextMap[wr.Conf.This] != nil {
		next = []*pb.Blueprint{cs.NextMap[wr.Conf.This]}
	}
	
	if wr.Conf.Cur < cs.CurC {
		return &pb.WriteSReply{Cur: &pb.ConfReply{cs.Cur, false}, Next: next}, nil
	}
	return &pb.WriteSReply{Next: next}, nil
}

func (cs *ConsServer) AWriteN(ctx context.Context, wr *pb.WriteN) (*pb.WriteNReply, error) {
	cs.Lock()
	defer cs.Unlock()
	glog.V(5).Infoln("Handling WriteN")
	
	if wr.CurC < cs.CurC {
		return &pb.WriteNReply{Cur: cs.Cur}, nil
	}
	
	cs.NextMap[wr.CurC] = wr.Next
	var next []*pb.Blueprint
	if wr.Next != nil {
		next = []*pb.Blueprint{wr.Next}
	}


	return &pb.WriteNReply{State: cs.RState, Next: next}, nil
}

func (cs *ConsServer) SetState(ctx context.Context, ns *pb.NewState) (*pb.NewStateReply, error) {
	cs.Lock()
	defer cs.Unlock()
	glog.V(5).Infoln("Handling SetState")
	if ns == nil {
		return nil, errors.New("Empty NewState message")
	}

	if cs.CurC > ns.CurC {
		return &pb.NewStateReply{Cur: cs.Cur}, nil
	}

	if cs.RState.Compare(ns.State) == 1 {
		cs.RState = ns.State
	}

	// The compare below is not necessary. But better safe than sorry.
	if cs.CurC < ns.CurC && cs.Cur.Compare(ns.Cur) == 1 {
		glog.V(3).Infoln("New Current Conf: ", ns.Cur)
		cs.Cur = ns.Cur
		cs.CurC = ns.CurC
	}

	var next []*pb.Blueprint
	if cs.NextMap[ns.CurC] != nil {
		next = []*pb.Blueprint{cs.NextMap[ns.CurC]}
	}
	return &pb.NewStateReply{Next: next}, nil
}