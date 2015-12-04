package main

import (
	"github.com/golang/glog"

	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
)

type FwdClient struct {
	RWRer
	leader *pb.Configuration
}

func (fc *FwdClient) Reconf(cp conf.Provider, prop *pb.Blueprint) (int, error) {
	_, err := fc.leader.Fwd(&pb.Proposal{prop})
	if err != nil {
		glog.Errorln("Forward returned error", err)
	}
	return 1, err
}
