package confProvider

import pb "github.com/relab/smartMerge/proto"

// This config provider does not avoid recontacting servers.
// Oups is only thrifty, if underlying provider is also thrifty.
type ThriftyConfP struct {
	Provider
}

func (cp *ThriftyConfP) ReadC(blp *pb.Blueprint, rids []int) *pb.Configuration {
	return cp.Provider.ReadC(blp, nil)
}

func (cp *ThriftyConfP) WriteC(blp *pb.Blueprint, rids []int) *pb.Configuration {
	return cp.Provider.WriteC(blp, nil)
}

func (cp *ThriftyConfP) GIDs(in []int) []uint32 {
	return nil
}

func (cp *ThriftyConfP) WriteCNoS(blp *pb.Blueprint, rids []int) *pb.Configuration {
	return cp.Provider.WriteCNoS(blp, nil)
}
