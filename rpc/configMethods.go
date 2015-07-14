package rpc

import (
	"errors"

	"golang.org/x/net/context"

	pb "github.com/relab/smartMerge/proto"
	lat "github.com/relab/smartMerge/directCombineLattice"
	"github.com/relab/smartMerge/regserver"
)

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
	replies, err :=c.mgr.ReadN(c.id, context.Background())
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
		if rNr != nil {
			for _,blp := range rNr.Next {
				bp := lat.GetBlueprint(*blp)
				blps = add(blps,bp)
			}
		}
	}
	return &blps
}

func add(bls []lat.Blueprint,bp lat.Blueprint) []lat.Blueprint {
	newbls := make([]lat.Blueprint,len(bls)+1)
	inserted := false
	for i := range newbls {
		switch {
		case inserted:
			newbls[i]=bls[i-1]
		case i == len(bls) && !inserted:
			newbls[i] = bp
		case bls[i].Compare(bp) == -1:
			newbls[i] = bp
			inserted = true
		default:
			if bp.Compare(bls[i]) == 1 {
				return bls
			}
			newbls[i] = bls[i]
		}
	}

	return newbls
}
