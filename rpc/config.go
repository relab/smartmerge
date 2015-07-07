package rpc

import (
	"fmt"
	"errors"

	"github.com/relab/smartMerge/regserver"
	pb "github.com/relab/smartMerge/proto"
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

func (c *Configuration) SRead() (pb.State, error) {
	s := regserver.InitState
	replies, err :=c.mgr.SRead(c.id, context.Background())
	if err != nil {
		return s, err
	}
	if len(replies) < 1 {
		return s, errors.New("No reply was returned.")
	}

	for _, st := range replies {
		if st != nil && st.Timestamp > s.Timestamp {
			s = *st
		}
	}
	return s,nil
}
