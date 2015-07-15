package rpc

import (
	"golang.org/x/net/context"

	pb "github.com/relab/smartMerge/proto"
	lat "github.com/relab/smartMerge/directCombineLattice"
	//"github.com/relab/smartMerge/regserver"
)

func (c *Configuration) ReadS(cur *lat.Blueprint) (s pb.State, newCur *lat.Blueprint,err error) {
	replies, newCur, err :=c.mgr.ReadS(c.id, cur, context.Background())
	if err != nil || newCur != nil  {
		return
	}

	for _, st := range replies {
		if st != nil && s.Compare(st.State)== 1 {
			s = *st.State
		}
	}
	return
}

func (c *Configuration) WriteS(s *pb.State, cur *lat.Blueprint) (newCur *lat.Blueprint, err error) {
	return c.mgr.WriteS(c.id, cur, context.Background(), &pb.WriteRequest{s, c.id})
}

func (c *Configuration) ReadN(cur *lat.Blueprint) (next []lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.ReadN(c.id, cur, context.Background())
	if err != nil || newCur != nil {
		return
	}
	next = GetBlueprintSlice(replies)
	return
}

func (c *Configuration) WriteN(next *lat.Blueprint, cur *lat.Blueprint) (newCur *lat.Blueprint, err error) {
	bp := next.ToMsg()
	return c.mgr.WriteN(c.id, cur, context.Background(), &pb.WriteNRequest{c.id, &bp})
}

func GetBlueprintSlice(replies []*pb.ReadNReply) []lat.Blueprint {
	blps := make([]lat.Blueprint,0)
	for _, rNr := range replies {
		if rNr != nil {
			for _,blp := range rNr.Next {
				bp := lat.GetBlueprint(*blp)
				blps = add(blps,bp)
			}
		}
	}
	return blps
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
