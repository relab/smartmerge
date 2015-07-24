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

func (rs *DynaServer) SetCur(ctx context.Context, nc *pb.NewCur) (*pb.NewCurReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	defer rs.PrintState("SetCur")

	if nc.CurC == rs.CurC {
		return &pb.NewCurReply{false}, nil
	}

	if rs.CurC == 0 || lat.Compare(nc.Cur, rs.Cur) == 1 {
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
	defer rs.PrintState("readS")

	if rr.CurC != rs.CurC {
		//Not sure if we should return an empty Next and State in this case.
		//Returning it is safer. The other faster.
		return &pb.AdvReadReply{rs.RState, rs.Cur, rs.Next[rr.CurC]}, nil
	}

	return &pb.AdvReadReply{State: rs.RState, Next: rs.Next[rr.CurC]}, nil
}

func (rs *DynaServer) DWriteS(ctx context.Context, wr *pb.AdvWriteS) (*pb.AdvWriteSReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	defer rs.PrintState("writeS")
	if rs.RState.Compare(wr.State) == 1 {
		rs.RState = wr.State
	}

	if wr.CurC == 0 {
		return &pb.AdvWriteSReply{}, nil
	}

	if wr.CurC != rs.CurC {
		//Not sure if we should return an empty Next in this case.
		//Returning it is safer. The other faster.
		return &pb.AdvWriteSReply{rs.Cur, rs.Next[wr.CurC]}, nil
	}

	return &pb.AdvWriteSReply{Next: rs.Next[wr.CurC]}, nil
}

func (rs *DynaServer) DWriteNSet(ctx context.Context, wr *pb.DWriteN) (*pb.DWriteNReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	defer rs.PrintState("writeN")

	if rs.Next[wr.CurC] == nil {
		rs.Next[wr.CurC] = wr.Next
	}

	for i, newBp := range wr.Next {
		for _, bp := range rs.Next[wr.CurC] {
			if lat.Equals(bp, newBp) {
				wr.Next[i] = nil
				break
			}
		}
	}

	for _, newBp := range wr.Next {
		if newBp != nil {
			rs.Next[wr.CurC] = append(rs.Next[wr.CurC], newBp)
		}
	}

	if wr.CurC != rs.CurC {
		return &pb.DWriteNReply{Cur: rs.Cur}, nil
	}

	return &pb.DWriteNReply{}, nil
}

func (rs *DynaServer) GetOneN(ctx context.Context, gt *pb.GetOne) (gtr *pb.GetOneReply, err error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	defer rs.PrintState("LAProp")

	if len(rs.Next[gt.CurC]) == 0 {
		rs.Next[gt.CurC] = []*pb.Blueprint{gt.Next}
	}

	var c *pb.Blueprint
	if gt.CurC != rs.CurC {
		c = rs.Cur
	}

	return &pb.GetOneReply{Cur: c, Next: rs.Next[gt.CurC][0]}, nil
}
