package rpc

import (
	"errors"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
)

//Cur is used to check if some server returned a new current Blueprint.
//In this case, the call is aborted.
//If cur == nil, any returned Blueprint results in an abort.
func (m *Manager) AReadS(configID uint32, cur *lat.Blueprint, ctx context.Context, opts ...grpc.CallOption) ([]*pb.AdvReadReply, *lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan *pb.AdvReadReply, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		out        = make([]*pb.AdvReadReply, 0, c.ReadQuorumSize())
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.AdvReadReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/AReadS", &pb.AdvRead{configID}, repl, machine.conn, c.grpcCallOptions...)
			}()
			select {
			case err := <-ce:
				if err != nil {
					machine.lastErr = err
					errSignal <- true
					return
				}
				machine.latency = time.Since(start)
				replyChan <- repl
			case <-stopSignal:
				return
			}
		}(ma)
	}

	for {
		select {
		case r := <-replyChan:
			if r.Cur != nil {
				newCur := lat.GetBlueprint(r.Cur)
				if cur == nil {
					//Abort if any Cur returned
					return nil, newCur, nil
				}
				if cur.Compare(newCur) == 1 {
					//Abort only if new cur was returned.
					return nil, newCur, nil
				}
			}

			out = append(out, r)
			if len(out) >= c.ReadQuorumSize() {
				return out, nil, nil
			}

		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.ReadQuorumSize() {
				return nil, nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) AWriteS(configID uint32, cur *lat.Blueprint, ctx context.Context, args *pb.AdvWriteS, opts ...grpc.CallOption) ([]*pb.AdvWriteSReply, *lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, nil, ConfigNotFound(configID)
	}

	q := c.quorum
	if q < c.ReadQuorumSize() {
		q = c.ReadQuorumSize()
	}
	var (
		replyChan  = make(chan *pb.AdvWriteSReply, q)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, q)
		out        = make([]*pb.AdvWriteSReply, 0, q)
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.AdvWriteSReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/AWriteS", args, repl, machine.conn, c.grpcCallOptions...)
			}()
			select {
			case err := <-ce:
				if err != nil {
					machine.lastErr = err
					errSignal <- true
					return
				}
				machine.latency = time.Since(start)
				replyChan <- repl
			case <-stopSignal:
				return
			}
		}(ma)
	}

	for {
		select {
		case r := <-replyChan:
			if r.Cur != nil {
				newCur := lat.GetBlueprint(r.Cur)
				if cur == nil {
					//Abort if any Cur returned
					return nil, newCur, nil
				}
				if cur.Compare(newCur) == 1 {
					//Abort only if new cur was returned.
					return nil, newCur, nil
				}
			}

			out = append(out, r)
			if len(out) >= q {
				return out, nil, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-q {
				return nil, nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) LAProp(configID uint32, cur *lat.Blueprint, ctx context.Context, args *pb.LAProposal, opts ...grpc.CallOption) ([]*pb.LAReply, *lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, nil, ConfigNotFound(configID)
	}

	q := c.quorum
	if q < c.ReadQuorumSize() {
		q = c.ReadQuorumSize()
	}
	var (
		replyChan  = make(chan *pb.LAReply, q)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, q)
		out        = make([]*pb.LAReply, 0, q)
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.LAReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/LAProp",
					args, repl, machine.conn, c.grpcCallOptions...)
			}()
			select {
			case err := <-ce:
				if err != nil {
					machine.lastErr = err
					errSignal <- true
					return
				}
				machine.latency = time.Since(start)
				replyChan <- repl
			case <-stopSignal:
				return
			}
		}(ma)
	}

	for {
		select {
		case r := <-replyChan:
			if r.Cur != nil {
				newCur := lat.GetBlueprint(r.Cur)
				if cur == nil {
					//Abort if any Cur returned
					return nil, newCur, nil
				}
				if cur.Compare(newCur) == 1 {
					//Abort only if new cur was returned.
					return nil, newCur, nil
				}
			}

			out = append(out, r)
			if len(out) >= q {
				return out, nil, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-q {
				return nil, nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) AWriteN(configID uint32, cur *lat.Blueprint, ctx context.Context, args *pb.AdvWriteN, opts ...grpc.CallOption) ([]*pb.AdvWriteNReply, *lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, nil, ConfigNotFound(configID)
	}

	q := c.quorum
	if q < c.ReadQuorumSize() {
		q = c.ReadQuorumSize()
	}
	var (
		replyChan  = make(chan *pb.AdvWriteNReply, q)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, q)
		out        = make([]*pb.AdvWriteNReply, 0, q)
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.AdvWriteNReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/AWriteN", args, repl, machine.conn, c.grpcCallOptions...)
			}()
			select {
			case err := <-ce:
				if err != nil {
					machine.lastErr = err
					errSignal <- true
					return
				}
				machine.latency = time.Since(start)
				replyChan <- repl
			case <-stopSignal:
				return
			}
		}(ma)
	}

	for {
		select {
		case r := <-replyChan:
			if r.Cur != nil {
				newCur := lat.GetBlueprint(r.Cur)
				if cur == nil {
					//Abort if any Cur returned
					return nil, newCur, nil
				}
				if cur.Compare(newCur) == 1 {
					//Abort only if newCur larger than current.
					return nil, newCur, nil
				}
			}

			out = append(out, r)
			if len(out) >= c.QuorumSize() {
				return out, nil, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-q {
				return nil, nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}
