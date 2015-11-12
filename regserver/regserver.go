package regserver

import (
	"errors"
	"fmt"
	"sync"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
)

type RegServer struct {
	sync.RWMutex
	Cur     *pb.Blueprint
	CurC    uint32
	LAState *pb.Blueprint //Used only for SM-Lattice agreement
	RState  *pb.State
	Next    []*pb.Blueprint
	NextMap map[uint32]*pb.Blueprint //Used only for Consensus based
	Rnd     map[uint32]uint32        //Used only for Consensus based
	Val     map[uint32]*pb.CV        //Used only for Consensus based
	noabort bool
}

func (rs *RegServer) PrintState(op string) {
	fmt.Println("Did operation :", op)
	fmt.Println("New State:")
	fmt.Println("Cur ", rs.Cur)
	fmt.Println("CurC ", rs.CurC)
	fmt.Println("LAState ", rs.LAState)
	fmt.Println("RState ", rs.RState)
	fmt.Println("Next", rs.Next)
}

var InitState = pb.State{Value: nil, Timestamp: int32(0), Writer: uint32(0)}

func NewRegServer(noabort bool) *RegServer {
	rs := &RegServer{}
	rs.RWMutex = sync.RWMutex{}
	rs.RState = &pb.State{make([]byte, 0), int32(0), uint32(0)}
	rs.Next = make([]*pb.Blueprint,0,5)
	rs.NextMap = make(map[uint32]*pb.Blueprint, 5)
	rs.Rnd = make(map[uint32]uint32, 5)
	rs.Val = make(map[uint32]*pb.CV, 5)
	rs.noabort = noabort
	return rs
}

func NewRegServerWithCur(cur *pb.Blueprint, curc uint32, noabort bool) *RegServer {
	rs := NewRegServer(noabort)
	rs.Cur = cur
	rs.CurC = curc

	return rs
}

// Used to set the current configuration. Currenlty only used at startup.
func (rs *RegServer) SetCur(ctx context.Context, nc *pb.NewCur) (*pb.NewCurReply, error) {
	glog.V(5).Infoln("Handling Set Cur")
	rs.Lock()
	defer rs.Unlock()
	//defer rs.PrintState("SetCur")

	if nc.CurC == rs.CurC {
		return &pb.NewCurReply{false}, nil
	}

	if nc.Cur.LearnedCompare(rs.Cur) >= 0 {
		return &pb.NewCurReply{false}, nil
	}

	// This could be removed. Not sure this is necessary.
	if rs.Cur.Compare(nc.Cur) == 0 {
		return &pb.NewCurReply{false}, errors.New("New Current Blueprint was uncomparable to previous.")
	}

	glog.V(3).Infoln("New Current Conf: ", nc.GetCur())
	rs.Cur = nc.Cur
	rs.CurC = nc.CurC

	newNext := make([]*pb.Blueprint, 0, len(rs.Next))
	for _, blp := range rs.Next {
		if blp.LearnedCompare(rs.Cur) == -1 {
			newNext = append(newNext, blp)
		}
	}
	rs.Next = newNext

	return &pb.NewCurReply{true}, nil
}

func (rs *RegServer) AReadS(ctx context.Context, rr *pb.Conf) (*pb.ReadReply, error) {
	rs.RLock()
	defer rs.RUnlock()
	glog.V(5).Infoln("Handling ReadS")

	if rr.This < rs.CurC && !rs.noabort {
		// The client is in an outdated configuration.
		return &pb.ReadReply{State: nil, Cur: &pb.ConfReply{rs.Cur, true}, Next: nil}, nil
	}

	next := make([]*pb.Blueprint, 0, len(rs.Next))
	this := int(rr.This)
	for _, nxt := range rs.Next {
		if nxt.Len() > this {
			next = append(next, nxt)
		}
	}

	if rr.Cur < rs.CurC {
		return &pb.ReadReply{State: rs.RState, Cur: &pb.ConfReply{rs.Cur, false}, Next: next}, nil
	}

	return &pb.ReadReply{State: rs.RState, Next: next}, nil
}

func (rs *RegServer) AWriteS(ctx context.Context, wr *pb.WriteS) (*pb.WriteSReply, error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling WriteS")
	if rs.RState.Compare(wr.State) == 1 {
		rs.RState = wr.State
	}

	if wr.Conf == nil || (wr.Conf.This < rs.CurC && !rs.noabort) {
		// The client is in an outdated configuration.
		return &pb.WriteSReply{Cur: &pb.ConfReply{rs.Cur, true}}, nil
	}
	next := make([]*pb.Blueprint, 0, len(rs.Next))
	this := int(wr.Conf.This)
	for _, nxt := range rs.Next {
		if nxt.Len() > this {
			next = append(next, nxt)
		}
	}
	if wr.Conf.Cur < rs.CurC {
		return &pb.WriteSReply{Cur: &pb.ConfReply{rs.Cur, false}, Next: next}, nil
	}
	return &pb.WriteSReply{Next: next}, nil
}

func (rs *RegServer) AWriteN(ctx context.Context, wr *pb.WriteN) (*pb.WriteNReply, error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling WriteN")

	var cur *pb.ConfReply
	if wr.CurC < rs.CurC {
		if !rs.noabort {
			return &pb.WriteNReply{Cur: &pb.ConfReply{rs.Cur,true}}, nil
		}
		cur = &pb.ConfReply{rs.Cur, false}
	}

	found := false
	for _, bp := range rs.Next {
		if bp.LearnedEquals(wr.Next) {
			found = true
			break
		}
	}
	if !found {
		rs.Next = append(rs.Next, wr.Next)
	}

	rs.NextMap[wr.CurC] = wr.Next

	next := make([]*pb.Blueprint, 0, len(rs.Next))
	this := int(wr.CurC)
	for _, nxt := range rs.Next {
		if nxt.Len() > this {
			next = append(next, nxt)
		}
	}

	return &pb.WriteNReply{Cur: cur, State: rs.RState, Next: next, LAState: rs.LAState}, nil
}

func (rs *RegServer) LAProp(ctx context.Context, lap *pb.LAProposal) (lar *pb.LAReply, err error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling LAProp")
	//defer rs.PrintState("LAProp")
	if lap == nil {
		return &pb.LAReply{Cur: &pb.ConfReply{rs.Cur,false}, LAState: rs.LAState, Next: rs.Next}, nil
	}

	if lap.Conf == nil || (lap.Conf.This < rs.CurC && !rs.noabort) {
			return &pb.LAReply{Cur: &pb.ConfReply{rs.Cur, true}}, nil
	}

	var c *pb.ConfReply
	if lap.Conf.Cur < rs.CurC {
		c = &pb.ConfReply{rs.Cur, false}
	}

	if rs.LAState.Compare(lap.Prop) == 1 {
		glog.V(6).Infoln("LAState Accepted")
		//Accept
		rs.LAState = lap.Prop
		next := make([]*pb.Blueprint, 0, len(rs.Next))
		this := int(lap.Conf.This)
		for _, nxt := range rs.Next {
			if nxt.Len() > this {
				next = append(next, nxt)
			}
		}
		return &pb.LAReply{Cur: c, Next: next}, nil
	}

	//Not Accepted, try again.
	rs.LAState = rs.LAState.Merge(lap.Prop)
	return &pb.LAReply{Cur: c, LAState: rs.LAState}, nil
}

func (rs *RegServer) SetState(ctx context.Context, ns *pb.NewState) (*pb.NewStateReply, error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling SetState")
	if ns == nil {
		return nil, errors.New("Empty NewState message")
	}

	rs.LAState = rs.LAState.Merge(ns.LAState)
	if rs.RState.Compare(ns.State) == 1 {
		rs.RState = ns.State
	}

	if rs.CurC > ns.CurC {
		return &pb.NewStateReply{Cur: rs.Cur}, nil
	}

	if rs.CurC < ns.CurC && rs.Cur.Compare(ns.Cur) == 1 {
		glog.V(3).Infoln("New Current Conf: ", ns.Cur)
		rs.Cur = ns.Cur
		rs.CurC = ns.CurC
		next := make([]*pb.Blueprint, 0, len(rs.Next))
		for _, nxt := range rs.Next {
			if nxt.Len() > rs.Cur.Len() {
				next = append(next, nxt)
			}
		}
		rs.Next = next
	}

	next := make([]*pb.Blueprint, len(rs.Next))
	copy(next, rs.Next)
	return &pb.NewStateReply{Next: next}, nil
}

func (rs *RegServer) GetPromise(ctx context.Context, pre *pb.Prepare) (*pb.Promise, error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling Prepare")

	if pre.CurC < rs.CurC {
		return &pb.Promise{Cur: rs.Cur}, nil
	}

	if rs.NextMap[pre.CurC] != nil {
		// Something was decided already
		return &pb.Promise{Dec: rs.NextMap[pre.CurC]}, nil
	}

	if rnd, ok := rs.Rnd[pre.CurC]; !ok || pre.Rnd > rnd {
		// A Prepare in a new and higher round.
		rs.Rnd[pre.CurC] = pre.Rnd
		return &pb.Promise{Val: rs.Val[pre.CurC]}, nil
	}

	return &pb.Promise{Rnd: rs.Rnd[pre.CurC], Val: rs.Val[pre.CurC]}, nil
}

func (rs *RegServer) Accept(ctx context.Context, pro *pb.Propose) (lrn *pb.Learn, err error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling Accept")

	if pro.CurC < rs.CurC {
		return &pb.Learn{Cur: rs.Cur}, nil
	}

	if rs.NextMap[pro.CurC] != nil {
		// This instance is decided already
		return &pb.Learn{Dec: rs.NextMap[pro.CurC]}, nil
	}

	if rs.Rnd[pro.CurC] > pro.Val.Rnd {
		// Accept in old round.
		return &pb.Learn{Learned: false}, nil
	}

	rs.Rnd[pro.CurC] = pro.Val.Rnd
	rs.Val[pro.CurC] = pro.Val
	return &pb.Learn{Learned: true}, nil
}
