package regserver

import (
	"sync"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
)

type SSRServer struct {
	Cur       *pb.Blueprint
	CurC      uint32 // This should be the length of cur, not its Gid.
	RState    *pb.State
	Proposed  map[uint32]map[uint32][]*pb.Blueprint //Conf, Rnd -> Proposals
	Committed map[uint32]map[uint32]*pb.Blueprint   //Conf, Rnd -> Committed value
	Collected map[uint32]map[uint32]*pb.Blueprint
	mu        sync.RWMutex
}

func NewSSRServer() *SSRServer {
	return &SSRServer{
		RState:    &pb.State{make([]byte, 0), int32(0), uint32(0)},
		Proposed:  make(map[uint32]map[uint32][]*pb.Blueprint, 5),
		Committed: make(map[uint32]map[uint32]*pb.Blueprint, 5),
		Collected: make(map[uint32]map[uint32]*pb.Blueprint, 5),
		mu:        sync.RWMutex{},
	}
}

func NewSSRServerWithCur(cur *pb.Blueprint, curc uint32) *SSRServer {
	srs := NewSSRServer()
	srs.Cur = cur
	srs.CurC = curc
	return srs
}

func (srs *SSRServer) SpSnOne(ctx context.Context, wn *pb.SWriteN) (*pb.SWriteNReply, error) {
	if wn.Prop == nil {
		srs.mu.RLock()
		defer srs.mu.RUnlock()
		glog.V(6).Infoln("handling empty SpSnOne")
		if wn.CurL < srs.CurC {
			return &pb.SWriteNReply{Cur: srs.Cur}, nil
		}
		proposed := srs.proposed(wn.This, wn.Rnd)

		if len(proposed) == 0 {
			return &pb.SWriteNReply{}, nil
		}
		return &pb.SWriteNReply{Next: proposed}, nil
	}

	srs.mu.Lock()
	defer srs.mu.Unlock()
	glog.V(5).Infoln("handling SpSnOne")

	if wn.CurL < srs.CurC {
		return &pb.SWriteNReply{Cur: srs.Cur}, nil
	}

	proposed := srs.proposed(wn.This, wn.Rnd)

	found := false
	for _, blp := range proposed {
		if blp.Equals(wn.Prop) {
			found = true
			break
		}
	}
	if !found {
		srs.Proposed[wn.This][wn.Rnd] = append(proposed, wn.Prop)
	}

	return &pb.SWriteNReply{Next: proposed}, nil
}

func (srs *SSRServer) proposed(this, rnd uint32) []*pb.Blueprint {
	if srs.Proposed[this] == nil {
		srs.Proposed[this] = make(map[uint32][]*pb.Blueprint, 1)
	}
	return srs.Proposed[this][rnd]
}

func (srs *SSRServer) SCommit(ctx context.Context, cm *pb.Commit) (*pb.CommitReply, error) {
	srs.mu.Lock()
	defer srs.mu.Unlock()
	glog.V(5).Infoln("handling SCommit")

	if cm.CurL < srs.CurC {
		return &pb.CommitReply{Cur: srs.Cur}, nil
	}

	if cm.Commit {
		if cm.Collect == nil {
			glog.Fatalln("Tried to commit an empty value.")
		}
		if srs.committed(cm.This, cm.Rnd) == nil {
			srs.Committed[cm.This][cm.Rnd] = cm.Collect
		} else if srs.committed(cm.This, cm.Rnd).Len() != cm.Collect.Len() {
			// The is a simple sanity check. It could be omitted.
			glog.Fatalln("Committing two different values in the same round.")
		}
		return &pb.CommitReply{Collected: srs.collected(cm.This, cm.Rnd)}, nil
	}
	x := srs.collected(cm.This, cm.Rnd)
	x = x.Merge(cm.Collect)
	srs.Collected[cm.This][cm.Rnd] = x
	return &pb.CommitReply{Collected: x, Committed: srs.committed(cm.This, cm.Rnd)}, nil
}

func (srs *SSRServer) committed(this, rnd uint32) *pb.Blueprint {
	if srs.Committed[this] == nil {
		srs.Committed[this] = make(map[uint32]*pb.Blueprint, 1)
	}
	return srs.Committed[this][rnd]
}

func (srs *SSRServer) collected(this, rnd uint32) *pb.Blueprint {
	if srs.Collected[this] == nil {
		srs.Collected[this] = make(map[uint32]*pb.Blueprint, 1)
	}
	return srs.Collected[this][rnd]
}

func (srs *SSRServer) SReadS(ctx context.Context, rd *pb.SRead) (*pb.SReadReply, error) {
	srs.mu.RLock()
	defer srs.mu.RUnlock()
	glog.V(5).Infoln("handling SRead")

	if rd.CurL < srs.CurC {
		return &pb.SReadReply{Cur: srs.Cur}, nil
	}

	return &pb.SReadReply{State: srs.RState}, nil
}

func (srs *SSRServer) SSetState(ctx context.Context, ss *pb.SState) (*pb.SStateReply, error) {
	srs.mu.Lock()
	defer srs.mu.Unlock()
	glog.V(5).Infoln("handling SSetState")

	var c *pb.Blueprint
	if ss.CurL < srs.CurC {
		c = srs.Cur
	}

	if srs.RState.Compare(ss.State) == 1 {
		srs.RState = ss.State
	}

	if srs.CurC < ss.CurL {
		srs.CurC = ss.CurL
		srs.Cur = ss.Cur
	}

	if len(srs.proposed(ss.CurL, 0)) != 0 {
		return &pb.SStateReply{HasNext: true, Cur: c}, nil
	}
	return &pb.SStateReply{HasNext: false, Cur: c}, nil
}
