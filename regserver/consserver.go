package regserver

import (
	"errors"
	"fmt"
	"sync"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
)

type ConsServer struct {
	Cur    *pb.Blueprint
	CurC   uint32
	RState *pb.State
	Next   map[uint32]*pb.Blueprint
	Rnd    map[uint32]uint32
	Val	   map[uint32]*pb.CV
	mu     sync.RWMutex
}

func (cs *ConsServer) PrintState(op string) {
	fmt.Println("Did operation :", op)
	fmt.Println("New State:")
	fmt.Println("Cur ", cs.Cur)
	fmt.Println("CurC ", cs.CurC)
	fmt.Println("RState ", cs.RState)
	fmt.Println("Next", cs.Next)
	fmt.Println("Rnd", cs.Rnd)
	fmt.Println("Val", cs.Val)
}

func NewConsServer() *ConsServer {
	return &ConsServer{
		RState: &pb.State{make([]byte, 0), int32(0), uint32(0)},
		Next:   make(map[uint32]*pb.Blueprint, 0),
		Rnd: 	make(map[uint32]uint32,0),
		Val:    make(map[uint32]*pb.CV,0),
		mu:     sync.RWMutex{},
	}
}

func NewConsServerWithCur(cur *pb.Blueprint, curc uint32) *ConsServer {
	return &ConsServer{
		Cur:    cur,
		CurC:   curc,
		RState: &pb.State{make([]byte, 0), int32(0), uint32(0)},
		Next:   make(map[uint32]*pb.Blueprint, 0),
		Rnd: 	make(map[uint32]uint32,0),
		Val:    make(map[uint32]*pb.CV,0),
		mu:     sync.RWMutex{},
	}
}

func (cs *ConsServer) CSetState(ctx context.Context, nc *pb.CNewCur) (*pb.NewCurReply, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	//defer cs.PrintState("SetCur")
	if cs.RState.Compare(nc.State) == 1 {
		cs.RState = nc.State
	}

	if nc.CurC == cs.CurC {
		return &pb.NewCurReply{false}, nil
	}

	if nc.CurC == 0 || lat.Compare(nc.Cur, cs.Cur) == 1 {
		return &pb.NewCurReply{false}, nil
	}

	if cs.Cur != nil && lat.Compare(cs.Cur, nc.Cur) == 0 {
		return &pb.NewCurReply{false}, errors.New("New Current Blueprint was uncomparable to previous.")
	}

	cs.Cur = nc.Cur
	cs.CurC = nc.CurC
	return &pb.NewCurReply{false}, nil
}

func (cs *ConsServer) CReadS(ctx context.Context, rr *pb.DRead) (*pb.AdvReadReply, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	//defer cs.PrintState("CReadS")

	if rr.Prop != nil {
		if n, ok := cs.Next[rr.CurC]; ok {
			if n != nil && !lat.Equals(n, rr.Prop) {
				return nil, errors.New("Tried to overwrite Next.")
			}
		} else {
			cs.Next[rr.CurC] = rr.Prop
		}
	}

	var next []*pb.Blueprint
	if cs.Next[rr.CurC] != nil {
		next = []*pb.Blueprint{cs.Next[rr.CurC]}
	}
	if rr.CurC != cs.CurC {
		//Not sure if we should return an empty Next and State in this case.
		//Returning it is safer. The other faster.
		return &pb.AdvReadReply{State: cs.RState,Cur: cs.Cur, Next: next}, nil
	}

	return &pb.AdvReadReply{State: cs.RState, Next: next}, nil
}

func (cs *ConsServer) CWriteS(ctx context.Context, wr *pb.AdvWriteS) (*pb.AdvWriteSReply, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	//defer cs.PrintState("CWriteS")
	if cs.RState.Compare(wr.State) == 1 {
		cs.RState = wr.State
	}

	if wr.CurC == 0 {
		return &pb.AdvWriteSReply{}, nil
	}

	var next []*pb.Blueprint
	if cs.Next[wr.CurC] != nil {
		next = []*pb.Blueprint{cs.Next[wr.CurC]}
	}
	
	if wr.CurC != cs.CurC {
		//Not sure if we should return an empty Next in this case.
		//Returning it is safer. The other faster.
		return &pb.AdvWriteSReply{Cur: cs.Cur,Next: next}, nil
	}

	return &pb.AdvWriteSReply{Next: next}, nil
}

func (cs *ConsServer) CPrepare(ctx context.Context, pre *pb.Prepare) (*pb.Promise, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	//defer cs.PrintState("CPrepare")

	var cur *pb.Blueprint
	if pre.CurC != cs.CurC {
		// Configuration outdated
		cur = cs.Cur
	}

	if cs.Next[pre.CurC] != nil {
		// Something was decided already
		return &pb.Promise{Cur: cur, Dec: cs.Next[pre.CurC]}, nil
	}

	if rnd, ok := cs.Rnd[pre.CurC]; !ok || pre.Rnd > rnd  {
		// A Prepare in a new and higher round.
		cs.Rnd[pre.CurC] = pre.Rnd 
		return &pb.Promise{Cur: cur, Val: cs.Val[pre.CurC]}, nil
	}
	
	return &pb.Promise{Cur: cur, Rnd: cs.Rnd[pre.CurC], Val: cs.Val[pre.CurC]}, nil
}

func (cs *ConsServer) CAccept(ctx context.Context, pro *pb.Propose) (lrn *pb.Learn, err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	//defer cs.PrintState("Accept")

	var cur *pb.Blueprint
	if pro.CurC != cs.CurC {
		// Configuration outdated.
		cur = cs.Cur
	}

	if cs.Next[pro.CurC] != nil {
		// This instance is decided already
		return &pb.Learn{Cur: cur, Dec: cs.Next[pro.CurC]}, nil
	}

	if cs.Rnd[pro.CurC] > pro.Val.Rnd {
		// Accept in old round.
		return &pb.Learn{Cur: cur, Learned: false}, nil
	}
	
	cs.Rnd[pro.CurC] = pro.Val.Rnd
	cs.Val[pro.CurC] = pro.Val
	return &pb.Learn{Cur: cur, Learned: true}, nil
}
