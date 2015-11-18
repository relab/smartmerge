package confProvider

import pb "github.com/relab/smartMerge/proto"

// This config provider does not avoid recontacting servers.
type ThriftyConfP struct {
	ThriftyNorecConfP
}

func (cp *ThriftyConfP) ReadC(blp *pb.Blueprint, rids []uint32) *pb.Configuration {
	return cp.ThriftyNorecConfP.ReadC(blp, nil)
}

func (cp *ThriftyConfP) WriteC(blp *pb.Blueprint, rids []uint32) *pb.Configuration {
		return cp.ThriftyNorecConfP.WriteC(blp, nil)
}

