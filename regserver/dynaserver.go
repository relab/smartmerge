package regserver

import (
	"errors"
	"fmt"
	"sync"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
)

type DynaServer struct {
	Cur    *pb.Blueprint
	CurC   uint32
	RState *pb.State
	Next   map[uint32][]*pb.Blueprint
	mu     sync.RWMutex
}

func (ds *DynaServer) PrintState(op string) {
	fmt.Println("Did operation :", op)
	fmt.Println("New State:")
	fmt.Println("Cur ", ds.Cur)
	fmt.Println("CurC ", ds.CurC)
	fmt.Println("RState ", ds.RState)
	fmt.Println("Next", ds.Next)
}

func NewDynaServer() *DynaServer {
	return &DynaServer{
		RState: &pb.State{make([]byte, 0), int32(0), uint32(0)},
		Next:   make(map[uint32][]*pb.Blueprint, 0),
		mu:     sync.RWMutex{},
	}
}

func NewDynaServerWithCur(cur *pb.Blueprint, curc uint32) *DynaServer {
	return &DynaServer{
		Cur:    cur,
		CurC:   curc,
		RState: &pb.State{make([]byte, 0), int32(0), uint32(0)},
		Next:   make(map[uint32][]*pb.Blueprint, 0),
		mu:     sync.RWMutex{},
	}
}

func (rs *DynaServer) DSetCur(ctx context.Context, nc *pb.NewCur) (*pb.NewCurReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	//defer rs.PrintState("SetCur")

	if nc.CurC == rs.CurC {
		return &pb.NewCurReply{false}, nil
	}

	if nc.CurC == 0 || lat.Compare(nc.Cur, rs.Cur) == 1 {
		return &pb.NewCurReply{false}, nil
	}

	if lat.Compare(rs.Cur, nc.Cur) == 0 {
		return &pb.NewCurReply{false}, errors.New("New Current Blueprint was uncomparable to previous.")
	}

	rs.Cur = nc.Cur
	rs.CurC = nc.CurC
	return &pb.NewCurReply{false}, nil
}

func (rs *DynaServer) DReadS(ctx context.Context, rr *pb.AdvRead) (*pb.AdvReadReply, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	//defer rs.PrintState("DReadS")

	var next []*pb.Blueprint
	if len(rs.Next[rr.CurC]) > 0 {
		next = rs.Next[rr.CurC]
	}
	if rr.CurC != rs.CurC {
		//Not sure if we should return an empty Next and State in this case.
		//Returning it is safer. The other faster.
		return &pb.AdvReadReply{State: rs.RState,Cur: rs.Cur, Next: next}, nil
	}

	return &pb.AdvReadReply{State: rs.RState, Next: next}, nil
}

func (rs *DynaServer) DWriteS(ctx context.Context, wr *pb.AdvWriteS) (*pb.AdvWriteSReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	//defer rs.PrintState("DWriteS")
	if rs.RState.Compare(wr.State) == 1 {
		rs.RState = wr.State
	}

	if wr.CurC == 0 {
		return &pb.AdvWriteSReply{}, nil
	}

	var next []*pb.Blueprint
	if len(rs.Next[wr.CurC]) > 0 {
		next = rs.Next[wr.CurC]
	}

	if wr.CurC != rs.CurC {
		//Not sure if we should return an empty Next in this case.
		//Returning it is safer. The other faster.
		return &pb.AdvWriteSReply{Cur: rs.Cur,Next: next}, nil
	}

	return &pb.AdvWriteSReply{Next: next}, nil
}

func (rs *DynaServer) DWriteNSet(ctx context.Context, wr *pb.DWriteN) (*pb.DWriteNReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	//defer rs.PrintState("DWriteNSet")

	if wr.Next == nil {
		return &pb.DWriteNReply{}, nil
	}

	if rs.Next[wr.CurC] == nil {
		rs.Next[wr.CurC] = wr.Next
	}
	outerLoop:
	for _, newBp := range wr.Next {
		for _, bp := range rs.Next[wr.CurC] {
			if lat.Equals(bp, newBp) {
				continue outerLoop
			}
		}
		rs.Next[wr.CurC] = append(rs.Next[wr.CurC], newBp)
	}

	if wr.CurC != rs.CurC {
		return &pb.DWriteNReply{Cur: rs.Cur}, nil
	}

	return &pb.DWriteNReply{}, nil
}

func (rs *DynaServer) GetOneN(ctx context.Context, gt *pb.GetOne) (gtr *pb.GetOneReply, err error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	//defer rs.PrintState("GetOneN")

	if len(rs.Next[gt.CurC]) == 0 {
		rs.Next[gt.CurC] = []*pb.Blueprint{gt.Next}
	}

	var c *pb.Blueprint
	if gt.CurC != rs.CurC {
		c = rs.Cur
	}
	
	return &pb.GetOneReply{Cur: c, Next: rs.Next[gt.CurC][0]}, nil
}

func (ds *DynaServer) CheckNext(curc uint32, op string) {
	if ds.Next[curc] == nil {
		return
	}
	for _,pb := range ds.Next[curc] {
		if pb == nil {
			fmt.Println("found nil in bp slice, doing ", op)
		}
	}
}