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
	for _,rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}
	return
}

func (c *Configuration) AWriteS(s *pb.State, cur *lat.Blueprint) (next []lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.AWriteS(c.id, cur, context.Background(), &pb.AdvWriteS{s, c.id})
	if err != nil || newCur != nil  {
		return
	}
	
	for _,rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}
	return
	
}

func (c *Configuration) LAProp(cur *lat.Blueprint, prop *lat.Blueprint) (las *lat.Blueprint, next []lat.Blueprint, newCur *lat.Blueprint, err error) {
	bp := prop.ToMsg()
	replies, newCur, err := c.mgr.LAProp(c.id, cur, context.Background(), &pb.LAProposal{c.id, &bp})
	if err != nil || newCur != nil {
		return
	}
	
	for _,rep := range replies {
		next = GetBlueprintSlice(next, rep)
		las = MergeLAState(las,rep)
	}
	return
}

func (c *Configuration) AWriteN(nnext *lat.Blueprint, cur *lat.Blueprint) (las *lat.Blueprint, next []lat.Blueprint, newCur *lat.Blueprint, err error) {
	bp := nnext.ToMsg()
	replies, newCur, err := c.mgr.AWriteN(c.id, cur, context.Background(), &pb.AdvWriteN{c.id, &bp})
	if err != nil || newCur != nil {
		return
	}

	for _,rep := range replies {
		next = GetBlueprintSlice(next, rep)
		las = MergeLAState(las,rep)
	}
	return
}

type LAStateReport interface {
	GetLAState() *pb.Blueprint
}

func MergeLAState(las *lat.Blueprint, rep LAStateReport) *lat.Blueprint {
	pb := rep.GetLAState()
	if pb == nil {
		return las
	}
	lap := lat.GetBlueprint(*pb)
	if las == nil {
		return &lap
	}
	newlat := las.Merge(lap)
	return &newlat
}