package consclient

import (
	"github.com/golang/glog"

	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
	smc "github.com/relab/smartMerge/smclient"
)

type ConsClient struct {
	*smc.SmClient
}

func New(initBlp *pb.Blueprint, id uint32, cp conf.Provider) (*ConsClient, error) {
	c, err := smc.New(initBlp, id, cp)
	if err != nil {
		return nil, err
	}
	return &ConsClient{c}, nil
}

func (cc *ConsClient) Reconf(cp conf.Provider, prop *pb.Blueprint) (cnt int, err error) {
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(cc.Blueps[0]) == 1 {
		glog.V(3).Infof("C%d: Proposal is already in place.", cc.Id)
		return 0, nil
	}

	_, cnt, err = cc.Doreconf(cp, prop, 0, nil)
	return
}
