package main

import (
	"golang.org/x/net/context"

	"github.com/golang/glog"

	bp "github.com/relab/smartMerge/blueprints"
	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
)

// The FwdClient implements a client, that, instead of performing reconfigurations
// forwards them to a leader.
type FwdClient struct {
	RWRer
	leader *pb.Node
}

// Reconf for the FwdClient simply forwards a reconfiguration request to the leader.
func (fc *FwdClient) Reconf(cp conf.Provider, prop *bp.Blueprint) (int, error) {
	if glog.V(4) {
		glog.Infoln("Sending reconfiguration proposal")
	}
	_, err := fc.leader.SMandConsRegisterClient.Fwd(context.Background(), &pb.Proposal{Prop: prop})
	if err != nil {
		glog.Errorln("Forward returned error", err)
	}
	if glog.V(4) {
		glog.Infoln("Proposal returned")
	}
	return 1, err
}
