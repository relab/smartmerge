package rpc

import (
	"errors"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
)

func (c *Configuration) ReadQuorumSize() int {
	return c.Size() - c.QuorumSize() + 1
}

//Cur is used to check if some server returned a new current Blueprint.
//In this case, the call is aborted.
//If cur == nil, any returned Blueprint results in an abort.
func (m *Manager) ReadS(configID uint32, cur *lat.Blueprint, ctx context.Context, opts ...grpc.CallOption) ([]*pb.ReadReply, *lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan *pb.ReadReply, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		out        = make([]*pb.ReadReply, 0, c.ReadQuorumSize())
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.ReadReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/ReadS", &pb.ReadRequest{configID}, repl, machine.conn, c.grpcCallOptions...)
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

func (m *Manager) WriteS(configID uint32, cur *lat.Blueprint, ctx context.Context, args *pb.WriteRequest, opts ...grpc.CallOption) (*lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan *pb.WriteReply, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		outCount   int
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.WriteReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/WriteS", args, repl, machine.conn, c.grpcCallOptions...)
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
					return newCur, nil
				}
				if cur.Compare(newCur) == 1 {
					//Abort only if new cur was returned.
					return newCur, nil
				}
			}

			outCount++
			if outCount >= c.QuorumSize() {
				return nil, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.ReadQuorumSize() {
				return nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) ReadN(configID uint32, cur *lat.Blueprint, ctx context.Context, opts ...grpc.CallOption) ([]*pb.ReadNReply, *lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan *pb.ReadNReply, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		out        = make([]*pb.ReadNReply, 0, c.ReadQuorumSize())
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.ReadNReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/ReadN",
					&pb.ReadNRequest{configID}, repl, machine.conn, c.grpcCallOptions...)
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

func (m *Manager) WriteN(configID uint32, cur *lat.Blueprint, ctx context.Context, args *pb.WriteNRequest, opts ...grpc.CallOption) (*lat.Blueprint, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan *pb.WriteNReply, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		outCount   int
		errCount   int
	)

	defer close(stopSignal)
	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.WriteNReply)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/WriteN", args, repl, machine.conn, c.grpcCallOptions...)
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
					return newCur, nil
				}
				if cur.Compare(newCur) == 1 {
					//Abort only if new cur was returned.
					return newCur, nil
				}
			}

			outCount++
			if outCount >= c.QuorumSize() {
				return nil, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.ReadQuorumSize() {
				return nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) SetCur(configID uint32, ctx context.Context, blp *pb.Blueprint, opts ...grpc.CallOption) ([]*pb.NewCurReply, error) {
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
				ce <- grpc.Invoke(ctx, "/proto.Register/WriteN", &pb.NewCur{blp, configID}, repl, machine.conn, c.grpcCallOptions...)
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
			if errCount > len(c.machines)-c.ReadQuorumSize() {
				return nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) SetCurASync(configID uint32, ctx context.Context, blp *pb.Blueprint, opts ...grpc.CallOption) error {
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
			grpc.Invoke(ctx, "/proto.Register/WriteN", &pb.NewCur{blp, configID}, repl, machine.conn, c.grpcCallOptions...)
		}(ma)
	}

	return nil
}
