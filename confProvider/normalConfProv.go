package confProvider

import (
	bp "github.com/relab/smartMerge/blueprints"
	pb "github.com/relab/smartMerge/proto"
)

// This Config provider always return a full configuration.
type NormalConfP struct {
	Provider
}

func (cp *NormalConfP) ReadC(blp *bp.Blueprint, rids []int) *pb.Configuration {
	return cp.Provider.FullC(blp)
}

func (cp *NormalConfP) WriteC(blp *bp.Blueprint, rids []int) *pb.Configuration {
	return cp.Provider.FullC(blp)
}

func (cp *NormalConfP) WriteCNoS(blp *bp.Blueprint, rids []int) *pb.Configuration {
	return cp.Provider.FullC(blp)
}

/*
func (cp *NormalConfP) SingleC(blp *pb.Blueprint) *pb.Configuration {
	return cp.Provider.ReadC(blp, nil)
}*/
