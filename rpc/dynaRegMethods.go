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
func (m *Manager) DReadS(configID uint32, cur *lat.Blueprint, ctx context.Context, opts ...grpc.CallOption) ([]*pb.AdvReadReply, *lat.Blueprint, error) {
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
				ce <- grpc.Invoke(ctx, "/proto.DynaDisk/DReadS", &pb.AdvRead{configID}, repl, machine.conn, c.grpcCallOptions...)
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
				if newCur.Compare(cur) != 1 {
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

func (m *Manager) DWriteS(configID uint32, cur *lat.Blueprint, ctx context.Context, args *pb.AdvWriteS, opts ...grpc.CallOption) ([]*pb.AdvWriteSReply, *lat.Blueprint, error) {
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
				ce <- grpc.Invoke(ctx, "/proto.DynaDisk/DWriteS", args, repl, machine.conn, c.grpcCallOptions...)
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
				if newCur.Compare(cur) != 1 {
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

func (m *Manager) GetOneN(configID uint32, cur *lat.Blueprint, ctx context.Context, prop *pb.Blueprint, opts ...grpc.CallOption) (*pb.GetOneReply, *lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, nil, ConfigNotFound(configID)
	}

	q := 1
	var (
		replyChan  = make(chan *pb.GetOneReply, q)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, q)
		errCount   int
	)

	defer close(stopSignal)
	max := c.quorum
	for _, mid := range c.machines {
		if max == 0 {
			continue
		}
		max--
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.GetOneReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.DynaDisk/GetOneN",
					&pb.GetOne{configID, prop}, repl, machine.conn, c.grpcCallOptions...)
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
				if newCur.Compare(cur) != 1 {
					//Abort only if new cur was returned.
					return nil, newCur, nil
				}
			}

			return r, nil, nil
		case <-errSignal:
			errCount++
			if errCount >= c.quorum {
				return nil, nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) DWriteNSet(configID uint32, cur *lat.Blueprint, ctx context.Context, args *pb.DWriteN, opts ...grpc.CallOption) ([]*pb.DWriteNReply, *lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, nil, ConfigNotFound(configID)
	}

	q := c.quorum
	var (
		replyChan  = make(chan *pb.DWriteNReply, q)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, q)
		out        = make([]*pb.DWriteNReply, 0, q)
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.DWriteNReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.DynaDisk/DWriteNSet", args, repl, machine.conn, c.grpcCallOptions...)
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
				if newCur.Compare(cur) != 1 {
					//Abort only if newCur larger than current.
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

func (m *Manager) DSetCur(configID uint32, ctx context.Context, blp *pb.Blueprint, opts ...grpc.CallOption) ([]*pb.NewCurReply, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan *pb.NewCurReply, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		out        = make([]*pb.NewCurReply, 0, c.quorum)
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.NewCurReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.DynaDisk/SetCur", &pb.NewCur{blp, configID}, repl, machine.conn, c.grpcCallOptions...)
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
			out = append(out, r)
			if len(out) >= c.QuorumSize() {
				return out, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.QuorumSize() {
				return nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) DSetCurASync(configID uint32, ctx context.Context, blp *pb.Blueprint, opts ...grpc.CallOption) error {
	c, found := m.configs[configID]
	if !found {
		return ConfigNotFound(configID)
	}

	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.NewCurReply)
			grpc.Invoke(ctx, "/proto.DynaDisk/SetCur", &pb.NewCur{blp, configID}, repl, machine.conn, c.grpcCallOptions...)
		}(ma)
	}

	return nil
}
