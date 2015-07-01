package regserver

import (
	"sync"
	
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
	lat "github.com/relab/smartMerge/directCombineLattice"
)

type RegServer struct {
	RState *pb.State
	Next []*pb.Blueprint
	mu sync.RWMutex
}

func NewRegServer() *RegServer {
	return &RegServer{
		RState : &pb.State{},
		Next : make([]*pb.Blueprint, 0),
		mu : sync.RWMutex{},
	}
}

func (rs *RegServer) ReadS(ctx context.Context, rr *pb.ReadRequest) ( *pb.State, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.RState, nil
}

func (rs *RegServer) ReadN(ctx context.Context, rr *pb.ReadNRequest) ( *pb.ReadNReply, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return &pb.ReadNReply{rs.Next}, nil
}

func (rs *RegServer) WriteS(ctx context.Context, s *pb.State) (*pb.WriteReply,error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if s.Timestamp > rs.RState.Timestamp {
		rs.RState = s
		return &pb.WriteReply{true}, nil
	}
	return &pb.WriteReply{false}, nil	
}

func (rs *RegServer) WriteN(ctx context.Context, bl *pb.Blueprint) (*pb.WriteNAck,error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	for _,bp := range rs.Next {
		if lat.Equals( *bp, *bl) {
			return &pb.WriteNAck{}, nil
		}
	}
	rs.Next = append(rs.Next, bl)
	return &pb.WriteNAck{}, nil
}