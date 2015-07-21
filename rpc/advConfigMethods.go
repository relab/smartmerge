package rpc

import (
	"golang.org/x/net/context"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	//"github.com/relab/smartMerge/regserver"
)

func (c *Configuration) AReadS(cur *lat.Blueprint) (s *pb.State, next []*lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.AReadS(c.id, cur, context.Background())
	if err != nil || newCur != nil {
		return
	}

	for _, rep := range replies {
		if s.Compare(rep.GetState()) == 1 {
			s = rep.GetState()
		}
	}
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
		newCur = CompareCur(newCur, rep)
	}
	return
}

func (c *Configuration) AWriteS(s *pb.State, thisBP *lat.Blueprint) (next []*lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.AWriteS(c.id, thisBP, context.Background(), &pb.AdvWriteS{s, c.id})
	if err != nil || newCur != nil {
		return
	}

	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
		newCur = CompareCur(newCur, rep)
	}
	return

}

func (c *Configuration) LAProp(thisBP *lat.Blueprint, prop *lat.Blueprint) (las *lat.Blueprint, next []*lat.Blueprint, newCur *lat.Blueprint, err error) {
	bp := prop.ToMsg()
	replies, newCur, err := c.mgr.LAProp(c.id, thisBP, context.Background(), &pb.LAProposal{c.id, bp})
	if err != nil || newCur != nil {
		return
	}

	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
		las = MergeLAState(las, rep)
		newCur = CompareCur(newCur, rep)
	}
	return
}

//TODO: This also has to return an RState.
func (c *Configuration) AWriteN(nnext *lat.Blueprint, thisBP *lat.Blueprint) (st *pb.State, las *lat.Blueprint, next []*lat.Blueprint, newCur *lat.Blueprint, err error) {
	bp := nnext.ToMsg()
	replies, newCur, err := c.mgr.AWriteN(c.id, thisBP, context.Background(), &pb.AdvWriteN{c.id, bp})
	if err != nil || newCur != nil {
		return
	}

	

	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
		las = MergeLAState(las, rep)
		newCur = CompareCur(newCur, rep)
		if st.Compare(rep.GetState()) == 1 {
			st = rep.GetState()
		}
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
	lap := lat.GetBlueprint(pb)
	if las == nil {
		return lap
	}
	return las.Merge(lap)
}

type CurReport interface {
	GetCur() *pb.Blueprint
}

func CompareCur(cur *lat.Blueprint,rep CurReport) *lat.Blueprint {
	newCur := lat.GetBlueprint(rep.GetCur())	
	if cur.Compare(newCur) == 1 {
		return newCur
	}
	return cur
}
