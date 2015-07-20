package rpc

import (
	"golang.org/x/net/context"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	//"github.com/relab/smartMerge/regserver"
)

type NextReport interface {
	GetNext() []*pb.Blueprint
}

func (c *Configuration) ReadS(cur *lat.Blueprint) (s *pb.State, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.ReadS(c.id, cur, context.Background())
	if err != nil || newCur != nil {
		return
	}

	for _, st := range replies {
		if s.Compare(st.State) == 1 {
			s = st.State
		}
	}
	return
}

func (c *Configuration) WriteS(s *pb.State, cur *lat.Blueprint) (newCur *lat.Blueprint, err error) {
	return c.mgr.WriteS(c.id, cur, context.Background(), &pb.WriteRequest{s, c.id})
}

func (c *Configuration) ReadN(cur *lat.Blueprint) (next []*lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.ReadN(c.id, cur, context.Background())
	if err != nil || newCur != nil {
		return
	}
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}
	return
}

func (c *Configuration) WriteN(next *lat.Blueprint, cur *lat.Blueprint) (newCur *lat.Blueprint, err error) {
	bp := next.ToMsg()
	return c.mgr.WriteN(c.id, cur, context.Background(), &pb.WriteNRequest{c.id, bp})
}

func (c *Configuration) SetCur(blp *lat.Blueprint) error {
	msgBlp := blp.ToMsg()

	_, err := c.mgr.SetCur(c.id, context.Background(), msgBlp)
	if err != nil {
		return err
	}
	return nil
}

func GetBlueprintSlice(next []*lat.Blueprint, rep NextReport) []*lat.Blueprint {
	for _, blp := range rep.GetNext() {
		bp := lat.GetBlueprint(blp)
		next = add(next, bp)
	}

	return next
}

func add(bls []*lat.Blueprint, bp *lat.Blueprint) []*lat.Blueprint {
	newbls := make([]*lat.Blueprint, len(bls)+1)
	inserted := false
	for i := range newbls {
		switch {
		case inserted:
			newbls[i] = bls[i-1]
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
