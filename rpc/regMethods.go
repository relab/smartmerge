 package rpc

import (
	"time"
	"errors"

	"google.golang.org/grpc"
	"golang.org/x/net/context"

	pb "github.com/relab/smartMerge/proto"
)

func (c *Configuration) ReadQuorumSize() int {
	return c.Size() - c.QuorumSize()
}

func (m *Manager) ReadS(configID uint32, ctx context.Context, opts ...grpc.CallOption) ([]*pb.State, error){
	c, found := m.configs[configID]
	if !found {
		return nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan *pb.State, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		out        = make([]*pb.State, c.quorum)
		errCount   int
	)

	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.State)
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/ReadS", &pb.ReadRequest{}, repl, machine.conn, c.grpcCallOptions...)
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
			if len(out) >= c.ReadQuorumSize() {
				close(stopSignal)
				return out, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.ReadQuorumSize() {
				close(stopSignal)
				return nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) WriteS(configID uint32, ctx context.Context, args *pb.State, opts ...grpc.CallOption) (error){
	c, found := m.configs[configID]
	if !found {
		return ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan bool, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		outCount   int
		errCount   int
	)

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
				replyChan <- true
			case <-stopSignal:
				return
			}
		}(ma)
	}

	for {
		select {
		case <-replyChan:
			outCount++
			if outCount >= c.QuorumSize() {
				close(stopSignal)
				return nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.ReadQuorumSize() {
				close(stopSignal)
				return errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) ReadN(configID uint32, ctx context.Context, opts ...grpc.CallOption) ([]*pb.ReadNReply, error){
	c, found := m.configs[configID]
	if !found {
		return nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan *ReadNReply, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		out        = make([]*pb.ReadNReply, c.quorum)
		errCount   int
	)

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
				ce <- grpc.Invoke(ctx, "/proto.Register/ReadN", &pb.ReadNRequest{}, repl, machine.conn, c.grpcCallOptions...)
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
			if len(out) >= c.ReadQuorumSize() {
				close(stopSignal)
				return out, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.ReadQuorumSize() {
				close(stopSignal)
				return nil, errors.New("could not complete request due to too many errors")
			}
		}
	}

}

func (m *Manager) WriteN(configID uint32, ctx context.Context, args *pb.Blueprint, opts ...grpc.CallOption) (error){
	c, found := m.configs[configID]
	if !found {
		return ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan bool, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		outCount   int
		errCount   int
	)

	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			repl := new(pb.WriteNAck)
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
				replyChan <- true
			case <-stopSignal:
				return
			}
		}(ma)
	}

	for {
		select {
		case <-replyChan:
			outCount++
			if outCount >= c.QuorumSize() {
				close(stopSignal)
				return nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.ReadQuorumSize() {
				close(stopSignal)
				return errors.New("could not complete request due to too many errors")
			}
		}
	}

}
