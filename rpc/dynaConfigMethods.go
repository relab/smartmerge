package rpc

import (
	"golang.org/x/net/context"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	//"github.com/relab/smartMerge/regserver"
)

func (c *Configuration) DReadS(thisBP *lat.Blueprint, prop *lat.Blueprint) (s *pb.State, next []*lat.Blueprint, newCur *lat.Blueprint, err error) {
	mprop := prop.ToMsg()
	replies, newCur, err := c.mgr.DReadS(c.id, thisBP, &pb.DRead{CurC: c.id, Prop: mprop}, context.Background())
	if err != nil || newCur != nil {
		return
	}

	for _, rep := range replies {
		if s.Compare(rep.GetState()) == 1 {
			s = rep.GetState()
		}
		next = GetBlueprintSlice(next, rep)
		newCur = CompareCur(newCur, rep)
	}
	return
}

func (c *Configuration) DWriteS(s *pb.State, thisBP *lat.Blueprint) (next []*lat.Blueprint, newCur *lat.Blueprint, err error) {
	replies, newCur, err := c.mgr.DWriteS(c.id, thisBP, context.Background(), &pb.AdvWriteS{s, c.id})
	if err != nil || newCur != nil {
		return
	}

	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
		newCur = CompareCur(newCur, rep)
	}
	return

}

func (c *Configuration) GetOneN(thisBP *lat.Blueprint, prop *lat.Blueprint) (next *lat.Blueprint, newCur *lat.Blueprint, err error) {
	bp := prop.ToMsg()
	reply, newCur, err := c.mgr.GetOneN(c.id, thisBP, context.Background(), bp)
	if err != nil || newCur != nil {
		return
	}

	next = lat.GetBlueprint(reply.Next)
	newCur = lat.GetBlueprint(reply.Cur)

	return
}

func (c *Configuration) DWriteNSet(nnext []*lat.Blueprint, thisBP *lat.Blueprint) (newCur *lat.Blueprint, err error) {
	mnext := make([]*pb.Blueprint, len(nnext))
	for i, bp := range nnext {
		mnext[i] = bp.ToMsg()
	}
	replies, newCur, err := c.mgr.DWriteNSet(c.id, thisBP, context.Background(), &pb.DWriteN{c.id, mnext})
	if err != nil || newCur != nil {
		return
	}

	for _, rep := range replies {
		newCur = CompareCur(newCur, rep)
	}

	return
}

func (c *Configuration) DSetCur(blp *lat.Blueprint) error {
	msgBlp := blp.ToMsg()

	_, err := c.mgr.DSetCur(c.id, context.Background(), msgBlp)
	if err != nil {
		return err
	}
	return nil
}
