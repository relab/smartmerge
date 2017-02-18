package regserver

import (
	"errors"
	"fmt"
	"sync"

	"github.com/golang/glog"
	bp "github.com/relab/smartMerge/blueprints"
	l "github.com/relab/smartMerge/leader"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
)

type RegServer struct {
	sync.RWMutex
	Cur     *bp.Blueprint            //Blueprint of the last installed configuration.
	CurC    uint32                   //Hash of the last installed configuration.
	LAState *bp.Blueprint            //Used only for SM-Lattice agreement
	RState  *pb.State                //Value timestamp stored.
	Next    []*bp.Blueprint          // A list of new blueprints
	NextMap map[uint32]*bp.Blueprint //Used only for Consensus based
	Rnd     map[uint32]uint32        //Used only for Consensus based
	Val     map[uint32]*pb.CV        //Used only for Consensus based
	noabort bool                     //
	Leader  *l.Leader
}

// PrintState prints the RegServers state, for debugging.
func (rs *RegServer) PrintState(op string) {
	fmt.Println("Did operation :", op)
	fmt.Println("New State:")
	fmt.Println("Cur ", rs.Cur)
	fmt.Println("CurC ", rs.CurC)
	fmt.Println("LAState ", rs.LAState)
	fmt.Println("RState ", rs.RState)
	fmt.Println("Next", rs.Next)
}

func NewRegServer(noabort bool) *RegServer {
	rs := &RegServer{}
	rs.RWMutex = sync.RWMutex{}
	rs.RState = &pb.State{Value: make([]byte, 0), Timestamp: int32(0), Writer: uint32(0)}
	rs.Next = make([]*bp.Blueprint, 0, 5)
	rs.NextMap = make(map[uint32]*bp.Blueprint, 5)
	rs.Rnd = make(map[uint32]uint32, 5)
	rs.Val = make(map[uint32]*pb.CV, 5)
	rs.noabort = noabort
	return rs
}

// NewRegServerWithCur creates a new RegServer with a specific initial configuration.
func NewRegServerWithCur(cur *bp.Blueprint, curc uint32, noabort bool) *RegServer {
	rs := NewRegServer(noabort)
	rs.Cur = cur
	rs.CurC = curc

	return rs
}

// handleConf updates the information about
// blueprints/ configuraitons stored at the server and returns
// all configurations larger than the current one.
func (rs *RegServer) handleConf(conf *pb.Conf, n *bp.Blueprint) (cr *pb.ConfReply) {
	if conf == nil || (conf.This < rs.CurC && !rs.noabort) {
		//The client is using an outdated configuration, abort.
		return &pb.ConfReply{Cur: rs.Cur, Abort: true}
	}

	if n != nil {
		found := false
		for _, nxt := range rs.Next {
			if n.LearnedEquals(nxt) {
				found = true
				break
			}
		}
		if !found {
			rs.Next = append(rs.Next, n)
		}
	}

	next := make([]*bp.Blueprint, 0, len(rs.Next))
	this := int(conf.This)
	for _, nxt := range rs.Next {
		if nxt.Len() > this {
			next = append(next, nxt)
		}
	}

	if conf.Cur < rs.CurC {
		// Inform the client of the new current configuration
		return &pb.ConfReply{Cur: rs.Cur, Abort: false, Next: next}
	}
	if len(next) > 0 {
		// Inform the client of the next configurations
		return &pb.ConfReply{Next: next}
	}
	return nil
}

func (rs *RegServer) Read(ctx context.Context, rr *pb.Conf) (*pb.ReadReply, error) {
	rs.RLock()
	defer rs.RUnlock()
	glog.V(5).Infoln("Handling ReadS")

	cr := rs.handleConf(rr, nil)
	if cr != nil && cr.Abort {
		return &pb.ReadReply{Cur: cr}, nil
	}

	return &pb.ReadReply{State: rs.RState, Cur: cr}, nil
}

// Write implements the Write RPC, that updates the stored register state.
// Also the list of blueprints is updated.
func (rs *RegServer) Write(ctx context.Context, wr *pb.WriteS) (*pb.ConfReply, error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling WriteS")

	// Update state, if new request has larger timestamp.
	if rs.RState.Compare(wr.GetState()) == 1 {
		rs.RState = wr.GetState()
	}

	if crepl := rs.handleConf(wr.GetConf(), nil); crepl != nil {
		return crepl, nil
	}
	return &pb.ConfReply{}, nil
}

// WriteNext implements the WriteNext RPC.
// WriteNext updates the list of blueprints stored at the server.
func (rs *RegServer) WriteNext(ctx context.Context, wr *pb.WriteN) (*pb.WriteNReply, error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling WriteN")

	cr := rs.handleConf(&pb.Conf{This: wr.CurC, Cur: wr.CurC}, wr.Next)
	if cr != nil && cr.Abort {
		return &pb.WriteNReply{Cur: cr}, nil
	}

	rs.NextMap[wr.CurC] = wr.Next // This is nor necessary for sm, but only for running Consensus using norecontact.

	return &pb.WriteNReply{Cur: cr, State: rs.RState, LAState: rs.LAState}, nil
}

// LAProp implements the LAProp RPC.
// LAProp merges a proposal with the current LAState and returns the result.
// this method also processes information about installed or learned blueprints.
func (rs *RegServer) LAProp(ctx context.Context, lap *pb.LAProposal) (lar *pb.LAReply, err error) {
	rs.Lock()
	defer rs.Unlock()
	glog.V(5).Infoln("Handling LAProp")

	cr := rs.handleConf(lap.GetConf(), nil)
	if cr != nil && cr.Abort {
		return &pb.LAReply{Cur: cr}, nil
	}

	if rs.LAState.Compare(lap.Prop) == 1 {
		glog.V(6).Infoln("LAState Accepted")
		//Accept
		rs.LAState = lap.Prop
		return &pb.LAReply{Cur: cr}, nil
	}

	//Not Accepted, try again.
	rs.LAState = rs.LAState.Merge(lap.Prop)
	if cr != nil {
		// In this case, we don't need to send the next values, since the client first has to solve LA in this configuration.
		cr.Next = nil
	}
	return &pb.LAReply{Cur: cr, LAState: rs.LAState}, nil
}

// SetState implements the SetState RPC.
// SetState updates the register and lattice agreement state.
// This method is used to transfer state to a new configuration.
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

	next := make([]*bp.Blueprint, 0, len(rs.Next))
	this := int(ns.CurC)
	for _, nxt := range rs.Next {
		if nxt.Len() > this {
			next = append(next, nxt)
		}
	}

	return &pb.NewStateReply{Next: next}, nil
}

// GetPromise implements the GetPromise RPC.
// This implements the first phase of Paxos.
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

// Accept implements the Accept RPC.
// This implements the second phase of Paxos.
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

// SetCur to inform about a new installed configuration.
func (rs *RegServer) SetCur(ctx context.Context, nc *pb.NewCur) (*pb.NewCurReply, error) {
	glog.V(5).Infoln("Handling Set Cur")
	rs.Lock()
	defer rs.Unlock()
	//defer rs.PrintState("SetCur")

	if nc.CurC == rs.CurC {
		return &pb.NewCurReply{New: false}, nil
	}

	if nc.Cur.LearnedCompare(rs.Cur) >= 0 {
		return &pb.NewCurReply{New: false}, nil
	}

	glog.V(3).Infoln("New Current Conf: ", nc.GetCur())
	rs.Cur = nc.Cur
	rs.CurC = nc.CurC

	newNext := make([]*bp.Blueprint, 0, len(rs.Next))
	for _, blp := range rs.Next {
		if uint32(blp.Len()) > rs.CurC {
			newNext = append(newNext, blp)
		}
	}
	rs.Next = newNext

	return &pb.NewCurReply{New: true}, nil
}

// Fwd implements the Fwd RPC.
// Fwd receives a reconfiguration request and forwards it to the leader (client)
// located at the same server.
func (rs *RegServer) Fwd(ctx context.Context, p *pb.Proposal) (*pb.Ack, error) {
	if rs.Leader == nil {
		glog.Errorln("Received Fwd request but have no leader.")
		return nil, errors.New("Not implemented.")
	}
	glog.V(4).Infoln("Handling Reconf Proposal")
	rs.Leader.Propose(p.GetProp())
	return &pb.Ack{}, nil
}

// AddLeader informs the server about a RPC client located at the same machine.
// This method must be invoked before any reconfiguration requests can be forwarded.
func (rs *RegServer) AddLeader(leader *l.Leader) {
	rs.Lock()
	defer rs.Unlock()
	rs.Leader = leader
}
