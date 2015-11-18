package confProvider

import pb "github.com/relab/smartMerge/proto"

// This Config provider always return a full configuration.
type NormalConfP struct {
	ThriftyNorecConfP
}

func (cp *NormalConfP) ReadC(blp *pb.Blueprint, rids []uint32) *pb.Configuration {
	return cp.ThriftyNorecConfP.FullC(blp)
}

func (cp *NormalConfP) WriteC(blp *pb.Blueprint, rids []uint32) *pb.Configuration {
		return cp.ThriftyNorecConfP.FullC(blp)
}

