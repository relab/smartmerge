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
	if glog.V(4) {
		glog.Infoln("Sending reconfiguration proposal")
	}
	_, err := fc.leader.Fwd(&pb.Proposal{prop})
	if err != nil {
		glog.Errorln("Forward returned error", err)
	}
	if glog.V(4) {
		glog.Infoln("Proposal returned")
	}
	return 1, err
}
