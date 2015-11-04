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
	Cur     *pb.Blueprint
	CurC    uint32
	LAState *pb.Blueprint
	RState  *pb.State
	Next    []*pb.Blueprint
	mu      sync.RWMutex
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

func NewRegServer() *RegServer {
	return &RegServer{
		LAState: new(pb.Blueprint),
		RState:  &pb.State{make([]byte, 0), int32(0), uint32(0)},
		Next:    make([]*pb.Blueprint, 0),
		mu:      sync.RWMutex{},
	}
}

func NewRegServerWithCur(cur *pb.Blueprint, curc uint32) *RegServer {
	return &RegServer{
		Cur:     cur,
		CurC:    curc,
		LAState: new(pb.Blueprint),
		RState:  &pb.State{make([]byte, 0), int32(0), uint32(0)},
		Next:    make([]*pb.Blueprint, 0),
		mu:      sync.RWMutex{},
	}
}

// func (rs *RegServer) ReadS(ctx context.Context, rr *pb.ReadRequest) (*pb.ReadReply, error) {
// 	rs.mu.RLock()
// 	defer rs.mu.RUnlock()
//
// 	if rr.CurC < rs.CurC {
// 		//Not sure if we should return an empty state in this case.
// 		//Returning it is safer. The other faster.
// 		return &pb.ReadReply{rs.RState, rs.Cur}, nil
// 	}
//
// 	return &pb.ReadReply{State: rs.RState}, nil
// }
//
// func (rs *RegServer) ReadN(ctx context.Context, rr *pb.ReadNRequest) (*pb.ReadNReply, error) {
// 	rs.mu.RLock()
// 	defer rs.mu.RUnlock()
//
// 	if rr.CurC < rs.CurC {
// 		//Not sure if we should return an empty Next in this case.
// 		//Returning it is safer. The other faster.
// 		return &pb.ReadNReply{rs.Cur, rs.Next}, nil
// 	}
//
// 	return &pb.ReadNReply{Next: rs.Next}, nil
// }
//
// func (rs *RegServer) WriteS(ctx context.Context, wr *pb.WriteRequest) (*pb.WriteReply, error) {
// 	rs.mu.Lock()
// 	defer rs.mu.Unlock()
// 	if rs.RState.Compare(wr.State) == 1 {
// 		rs.RState = wr.State
// 	}
//
// 	if wr.CurC < rs.CurC {
// 		return &pb.WriteReply{rs.Cur}, nil
// 	}
//
// 	return &pb.WriteReply{}, nil
// }
//
// func (rs *RegServer) WriteN(ctx context.Context, wr *pb.WriteNRequest) (*pb.WriteNReply, error) {
// 	rs.mu.Lock()
// 	defer rs.mu.Unlock()
// 	found := false
// 	for _, bp := range rs.Next {
// 		if lat.Equals(bp, (wr.Next)) {
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		rs.Next = append(rs.Next, wr.Next)
// 	}
//
// 	if wr.CurC < rs.CurC {
// 		return &pb.WriteNReply{rs.Cur}, nil
// 	}
//
// 	return &pb.WriteNReply{}, nil
// }

func (rs *RegServer) SetCur(ctx context.Context, nc *pb.NewCur) (*pb.NewCurReply, error) {
	glog.V(5).Infoln("Handling Set Cur")
	rs.mu.Lock()
	defer rs.mu.Unlock()
	//defer rs.PrintState("SetCur")

	if nc.CurC == rs.CurC {
		return &pb.NewCurReply{false}, nil
	}

	if nc.Cur.LearnedCompare(rs.Cur) == 1 {
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
		if blp.Compare(rs.Cur) == -1 {
			newNext = append(newNext, blp)
		}
	}
	rs.Next = newNext

	return &pb.NewCurReply{true}, nil
}

func (rs *RegServer) AReadS(ctx context.Context, rr *pb.Conf) (*pb.ReadReply, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	glog.V(5).Infoln("Handling ReadS")
	//defer rs.PrintState("readS")

	if rr.This < rs.CurC {
		//Not sure if we should return an empty Next and State in this case.
		//Returning it is safer. The other faster.
		return &pb.ReadReply{State: nil, Cur: &pb.ConfReply{rs.Cur, true}, Next: nil}, nil
	}
	if rr.Cur < rs.CurC {
		return &pb.ReadReply{State: rs.RState, Cur: &pb.ConfReply{rs.Cur, false}, Next: rs.Next}, nil
	}
	next := make([]*pb.Blueprint, 0, len(rs.Next))
	this := int(rr.This)
	for _, nxt := range rs.Next {
		if nxt.Len() > this {
			next = append(next, nxt)
		}
	}

	return &pb.ReadReply{State: rs.RState, Next: next}, nil
}

func (rs *RegServer) AWriteS(ctx context.Context, wr *pb.WriteS) (*pb.WriteSReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	glog.V(5).Infoln("Handling WriteS")
	//defer rs.PrintState("writeS")
	if rs.RState.Compare(wr.State) == 1 {
		rs.RState = wr.State
	}

	// if wr.CurC == 0 {
	// 		return &pb.AdvWriteSReply{}, nil
	// 	}

	if wr.Conf.This < rs.CurC {
		//Not sure if we should return an empty Next in this case.
		//Returning it is safer. The other faster.
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

func (rs *RegServer) AWriteN(ctx context.Context, wr *pb.AdvWriteN) (*pb.AdvWriteNReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	glog.V(5).Infoln("Handling WriteN")
	//defer rs.PrintState("writeN")
	found := false

	for _, bp := range rs.Next {
		if bp.Equals(wr.Next) {
			found = true
			break
		}
	}
	if !found {
		rs.Next = append(rs.Next, wr.Next)
	}

	if wr.CurC < rs.CurC {
		//Not sure if we should return an empty Next/State in this case.
		//Returning it is safer. The other faster.
		return &pb.AdvWriteNReply{Cur: rs.Cur, State: rs.RState, Next: rs.Next, LAState: rs.LAState}, nil
	}

	return &pb.AdvWriteNReply{State: rs.RState, Next: rs.Next, LAState: rs.LAState}, nil
}

func (rs *RegServer) LAProp(ctx context.Context, lap *pb.LAProposal) (lar *pb.LAReply, err error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	glog.V(5).Infoln("Handling LAProp")
	//defer rs.PrintState("LAProp")
	if lap == nil {
		return &pb.LAReply{Cur: rs.Cur, LAState: rs.LAState, Next: rs.Next}, nil
	}

	var c *pb.Blueprint
	if lap.CurC < rs.CurC {
		c = rs.Cur
	}

	if rs.LAState.Compare(lap.Prop) == 1 {
		glog.V(6).Infoln("LAState Accepted")
		//Accept
		rs.LAState = lap.Prop
		return &pb.LAReply{Cur: c, Next: rs.Next}, nil
	}

	//Not Accepted, try again.
	rs.LAState = rs.LAState.Merge(lap.Prop)
	return &pb.LAReply{Cur: c, LAState: rs.LAState}, nil
}

func (rs *RegServer) SetState(ctx context.Context, ns *pb.NewState) (*pb.NewStateReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	glog.V(5).Infoln("Handling SetState")
	if ns == nil {
		return nil, errors.New("Empty NewState message")
	}

	rs.LAState = rs.LAState.Merge(ns.LAState)
	if rs.RState.Compare(ns.State) == 1 {
		rs.RState = ns.State
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
		return &pb.NewStateReply{Next: rs.Next}, nil
	}

	if rs.CurC == ns.CurC {
		return &pb.NewStateReply{Next: rs.Next}, nil
	}
	return &pb.NewStateReply{Cur: rs.Cur, Next: rs.Next}, nil
}
