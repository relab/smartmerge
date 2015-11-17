package regserver

import (
	"errors"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
)

type ConsServer struct {
	*RegServer
}

func NewConsServer(noabort bool) *ConsServer {
	return &ConsServer{
		NewRegServer(noabort),
	}
}

func NewConsServerWithCur(cur *pb.Blueprint, curc uint32, noabort bool) *ConsServer {
	return &ConsServer{
		NewRegServerWithCur(cur, curc, noabort),
	}
}

func (cs *ConsServer) handleConf(conf *pb.Conf) (cr *pb.ConfReply) {
	if conf == nil || (conf.This < cs.CurC && !cs.noabort) {
		//The client is using an outdated configuration, abort.
		return &pb.ConfReply{Cur: cs.Cur, Abort: false}
	}

	if conf.Cur < cs.CurC {
		// Inform the client of the new current configuration
		return &pb.ConfReply{Cur: cs.Cur, Abort: false, Next: []*pb.Blueprint{cs.NextMap[conf.This]}}
	}
	if n:= cs.NextMap[conf.This]; n != nil {
		// Inform the client of the next configurations
		return &pb.ConfReply{Next: []*pb.Blueprint{n}}
	}
	return nil
}

func (cs *ConsServer) AReadS(ctx context.Context, rr *pb.Conf) (*pb.ReadReply, error) {
	cs.RLock()
	defer cs.RUnlock()
	glog.V(5).Infoln("Handling ReadS")

	cr := cs.handleConf(rr)
	if cr != nil && cr.Abort {
		return &pb.ReadReply{Cur: cr}, nil
	}

	return &pb.ReadReply{State: cs.RState, Cur: cr}, nil
}

func (cs *ConsServer) AWriteS(ctx context.Context, wr *pb.WriteS) (*pb.ConfReply, error) {
	cs.Lock()
	defer cs.Unlock()
	glog.V(5).Infoln("Handling WriteS")
	if cs.RState.Compare(wr.GetState()) == 1 {
		cs.RState = wr.GetState()
	}

	return cs.handleConf(wr.GetConf()), nil
}

func (cs *ConsServer) AWriteN(ctx context.Context, wr *pb.WriteN) (*pb.WriteNReply, error) {
	cs.Lock()
	defer cs.Unlock()
	glog.V(5).Infoln("Handling WriteN")

	cr := cs.handleConf(&pb.Conf{wr.CurC, wr.CurC})
	if cr != nil && cr.Abort {
		return &pb.WriteNReply{Cur: cr}, nil
	}

	cs.NextMap[wr.CurC] = wr.Next

	return &pb.WriteNReply{Cur: cr, State: cs.RState, LAState: cs.LAState}, nil
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
		for nc, _ := range cs.NextMap {
			if nc < cs.CurC {
				delete(cs.NextMap, nc)
			}
		}

	}

	var next []*pb.Blueprint
	if cs.NextMap[ns.CurC] != nil {
		next = []*pb.Blueprint{cs.NextMap[ns.CurC]}
	}
	return &pb.NewStateReply{Next: next}, nil
}
