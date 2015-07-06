package rpc

import (
	"fmt"

	pb "github.com/relab/grpc-test/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type MachineNotFound uint32

func (e MachineNotFound) Error() string {
	return fmt.Sprintf("machine not found: %d", e)
}

type ConfigNotFound uint32

func (e ConfigNotFound) Error() string {
	return fmt.Sprintf("configuration not found: %d", e)
}

type Configuration struct {
	id              uint32
	machines        []uint32
	quorum          int
	grpcCallOptions []grpc.CallOption
	mgr             *Manager
}

func (c *Configuration) ID() uint32 {
	return c.id
}

func (c *Configuration) Machines() []uint32 {
	return c.machines
}

func (c *Configuration) QuorumSize() int {
	return c.quorum
}

func (c *Configuration) Size() int {
	return len(c.machines)
}

func (c *Configuration) String() string {
	return fmt.Sprintf("Configuration %d", c.id)
}

func (c *Configuration) Equals(config *Configuration) bool {
	return c.id == config.id
}

// TODO(tormod): Everything below should be generated.
// This is just an example method (write for a register).

func (c *Configuration) Read(ctx context.Context, in *pb.ReadRequest) ([]*pb.State, error) {
	replies, err := c.mgr.invoke(c.id, ctx, "/proto.Register/Read", in, c.grpcCallOptions...)
	if err != nil {
		return nil, err
	}
	// TODO(tormod): Below - think about this
	outAsType := make([]*pb.State, len(c.machines))
	for i, r := range replies {
		outAsType[i] = r.reply.(*pb.State)
	}
	return outAsType, nil
}

func (c *Configuration) Write(ctx context.Context, in *pb.State) ([]*pb.WriteReply, error) {
	replies, err := c.mgr.invoke(c.id, ctx, "/proto.Register/Write", in, c.grpcCallOptions...)
	if err != nil {
		return nil, err
	}
	// TODO(tormod): Below - think about this
	outAsType := make([]*pb.WriteReply, len(c.machines))
	for i, r := range replies {
		outAsType[i] = r.reply.(*pb.WriteReply)
	}
	return outAsType, nil
}
