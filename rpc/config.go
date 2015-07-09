package rpc

import (
	"fmt"
	"errors"

	"github.com/relab/smartMerge/regserver"
	pb "github.com/relab/smartMerge/proto"
	lat "github.com/relab/smartMerge/directCombineLattice"

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

func (c *Configuration) ReadS() (pb.State, error) {
	s := regserver.InitState
	replies, err :=c.mgr.ReadS(c.id, context.Background())
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

func (c *Configuration) WriteS(s *pb.State) error {
	return c.mgr.WriteS(c.id, context.Background(), s)
}

func (c *Configuration) ReadN() ([]lat.Blueprint, error) {
	blps := make([]lat.Blueprint,0)
	replies, err :=c.mgr.ReadS(c.id, context.Background())
	if err != nil {
		return blps, err
	}
	if len(replies) < 1 {
		return blps, errors.New("No reply was returned.")
	}

	return *GetBlueprintSlice(replies), nil
}

func (c *Configuration) WriteN(b *lat.Blueprint) error {
	bp := b.ToMsg()
	return c.mgr.WriteN(c.id, context.Background(), &bp)
}

func GetBlueprintSlice(replies []*pb.ReadNReply) *[]lat.Blueprint {
	blps := make([]lat.Blueprint,0)
	for _, rNr := range replies {
		for _,blp := range rNr.Next {
			bp := lat.GetBlueprint(blp)
			blps = add(blps,bp)
		}
	}
	return &blps
}

func add(bls []lat.Blueprint,bp lat.Blueprint) []lat.Blueprint {
	newbls := make([]lat.Blueprint,len(lat.Blueprint)+1,len(lat.Blueprint)+1)
	inserted := false
	for i,b := range bls {
		switch {
		case inserted:
			newbls[i+1]=bls[i]
		case b.Compare(bp) == -1:
			newbls[i] = bp
			inserted = true
		case b.Compare(bp) != -1:
			if bp.Compare(b) == 1 {
				return bls
			}
			newbls[i] = bls[i]
		}
	}
	return newbls
}
