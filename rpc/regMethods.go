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

func (m *Manager) SRead(configID uint32, ctx context.Context, args interface{}, opts ...grpc.CallOption) ([]*pb.State, error){
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
			//r := rpcReply{mid: machine.id}
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, "/proto.Register/ReadS", args, repl, machine.conn, c.grpcCallOptions...)
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


