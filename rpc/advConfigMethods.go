package rpc

import (
	"golang.org/x/net/context"

	pb "github.com/relab/smartMerge/proto"
	lat "github.com/relab/smartMerge/directCombineLattice"
	//"github.com/relab/smartMerge/regserver"
)

func (c *Configuration) AReadS(cur *lat.Blueprint) (s pb.State, next []lat.Blueprint, newCur *lat.Blueprint,err error) {
	replies, newCur, err :=c.mgr.AReadS(c.id, cur, context.Background())
	if err != nil || newCur != nil  {
		return
	}

	for _, st := range replies {
		if st != nil && s.Compare(st.State)== 1 {
			s = *st.State
		}
	}
	next = GetBlueprintSlice(replies)
	return
}

func (c *Configuration) AWriteS(s *pb.State, cur *lat.Blueprint) (next []lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.AWriteS(c.id, cur, context.Background(), &pb.AdvWriteS{s, c.id})
	if err != nil || newCur != nil  {
		return
	}
	
	next = GetBlueprintSlice(replies)
	return
	
}

func (c *Configuration) ReadN(cur *lat.Blueprint) (next []lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.ReadN(c.id, cur, context.Background())
	if err != nil || newCur != nil {
		return
	}
	
	next = GetBlueprintSlice(replies)
	return
}

func (c *Configuration) AWriteN(nnext *lat.Blueprint, cur *lat.Blueprint) (las *lat.Blueprint, next []lat.Blueprint, newCur *lat.Blueprint, err error) {
	bp := next.ToMsg()
	replies, newCur, err := c.mgr.AWriteN(c.id, cur, context.Background(), &pb.AdvWriteN{c.id, &bp})
	if err != nil || newCur != nil {
		return
	}

	next = GetBlueprintSlice(replies)
	las = GetLAState(replies)
	return
}

func GetLAState(reps []*pb.AdvWriteNReply) *lat.Blueprint {
	b := New(pb.Blueprint)
	for _,las := range reps {
		//unfinished
	}
	
}