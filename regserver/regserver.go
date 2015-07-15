package regserver

import (
	"sync"
	"errors"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
)

type RegServer struct {
	Cur	   *pb.Blueprint
	CurC   uint32
	RState *pb.State
	Next   []*pb.Blueprint
	mu     sync.RWMutex
}

var InitState = pb.State{Value : nil, Timestamp : int32(0), Writer : uint32(0)}

func NewRegServer() *RegServer {
	return &RegServer{
		RState: &pb.State{make([]byte,0), int32(0), uint32(0)},
		Next:   make([]*pb.Blueprint, 0),
		mu:     sync.RWMutex{},
	}
}

func NewRegServerWithCur(cur *pb.Blueprint,curc uint32) *RegServer {
	return &RegServer{
		Cur: 	cur,
		CurC:  	curc,
		RState: &pb.State{make([]byte,0), int32(0), uint32(0)},
		Next:   make([]*pb.Blueprint, 0),
		mu:     sync.RWMutex{},
	}
}

func (rs *RegServer) ReadS(ctx context.Context, rr *pb.ReadRequest) (*pb.ReadReply, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	
	if rr.CurC != rs.CurC {
		//Not sure if we should return an empty state in this case. 
		//Returning it is safer. The other faster.
		return &pb.ReadReply{rs.RState,rs.Cur}, nil
	}
	
	return &pb.ReadReply{State : rs.RState}, nil
}

func (rs *RegServer) ReadN(ctx context.Context, rr *pb.ReadNRequest) (*pb.ReadNReply, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	
	if rr.CurC != rs.CurC {
		//Not sure if we should return an empty Next in this case. 
		//Returning it is safer. The other faster.
		return &pb.ReadNReply{rs.Cur,rs.Next}, nil
	}
	
	return &pb.ReadNReply{Next : rs.Next}, nil
}

func (rs *RegServer) WriteS(ctx context.Context, wr *pb.WriteRequest) (*pb.WriteReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.RState.Compare(wr.State) == 1 {
		rs.RState = wr.State
	} 

	if wr.CurC != rs.CurC {
		return &pb.WriteReply{rs.Cur}, nil
	}
	
	return &pb.WriteReply{}, nil
}

func (rs *RegServer) WriteN(ctx context.Context, wr *pb.WriteNRequest) (*pb.WriteNReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	found := false
	for _, bp := range rs.Next {
		if lat.Equals(*bp, *(wr.Next)) {
			found = true
			break
		}
	}
	if !found {
			rs.Next = append(rs.Next, wr.Next)
	}
	
	if wr.CurC != rs.CurC {
		return &pb.WriteNReply{rs.Cur}, nil
	}
	
	return &pb.WriteNReply{}, nil
}

func (rs *RegServer) SetCur(ctx context.Context, nc *pb.NewCur) (*pb.NewCurReply, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	if nc.CurC == rs.CurC {
		return &pb.NewCurReply{false}, nil
	}
	
	if rs.CurC == 0 || lat.Compare(*rs.Cur, *nc.Cur) == 1 {
		rs.Cur = nc.Cur
		rs.CurC = nc.CurC
		return &pb.NewCurReply{true}, nil
	}
	
	if lat.Compare(*rs.Cur, *nc.Cur) == 0 {
		return &pb.NewCurReply{false}, errors.New("New Current Blueprint was uncomparable to previous.")
	}
	
	return &pb.NewCurReply{false}, nil
}

func (rs *RegServer) AReadS(ctx context.Context, rr *pb.AdvRead) (*pb.AdvReadReply, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	
	if rr.CurC != rs.CurC {
		//Not sure if we should return an empty Next and State in this case. 
		//Returning it is safer. The other faster.
		return &pb.AdvReadReply{rs.RState,rs.Cur,rs.Next}, nil
	}
	
	return &pb.AdvReadReply{State : rs.RState, Next : rs.Next}, nil
}

func (rs *RegServer) AWriteS(ctx context.Context, wr *pb.AdvWriteS) (*pb.AdvWriteSReply, error) {
		rs.mu.Lock()
		defer rs.mu.Unlock()
		if rs.RState.Compare(wr.State) == 1 {
			rs.RState = wr.State
		}

		if wr.CurC != rs.CurC {
			//Not sure if we should return an empty Next in this case. 
			//Returning it is safer. The other faster.
			return &pb.AdvWriteSReply{rs.Cur, rs.Next}, nil
		}
	
		return &pb.AdvWriteSReply{Next : rs.Next}, nil
}

func (rs *RegServer) AWriteN(ctx context.Context, wr *pb.AdvWriteN) (*pb.AdvWriteNReply, error) {
		rs.mu.Lock()
		defer rs.mu.Unlock()
		found := false
		
		for _, bp := range rs.Next {
			if lat.Equals(*bp, *(wr.Next)) {
				found = true
				break
			}
		}
		if !found {
				rs.Next = append(rs.Next, wr.Next)
		}
	
		if wr.CurC != rs.CurC {
			//Not sure if we should return an empty Next/State in this case. 
			//Returning it is safer. The other faster.
			return &pb.AdvWriteNReply{rs.Cur, rs.RState, rs.Next}, nil
		}
	
		return &pb.AdvWriteNReply{State : rs.RState, Next : rs.Next}, nil
	}
