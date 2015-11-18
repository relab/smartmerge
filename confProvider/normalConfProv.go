package confProvider

import pb "github.com/relab/smartMerge/proto"

// This Config provider always return a full configuration.
type NormalConfP struct {
	Provider
}

func (cp *NormalConfP) ReadC(blp *pb.Blueprint, rids []int) *pb.Configuration {
	return cp.Provider.FullC(blp)
}

func (cp *NormalConfP) WriteC(blp *pb.Blueprint, rids []int) *pb.Configuration {
	return cp.Provider.FullC(blp)
}
