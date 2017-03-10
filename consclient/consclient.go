// Package consclient implements a register client,
// using consensus/Paxos to agree on new configurations.
// Read and Write operations are performed in the same way as in the
// smclient.
package consclient

import (
	"github.com/golang/glog"

	bp "github.com/relab/smartmerge/blueprints"
	conf "github.com/relab/smartmerge/confProvider"
	smc "github.com/relab/smartmerge/smclient"
)

// ConsClient wraps a SmClient.
// Only reconfiguration is implemented separately.
// Read and Write methods are perfomed as in smartmerge
type ConsClient struct {
	*smc.SmClient
}

func New(initBlp *bp.Blueprint, id uint32, cp conf.Provider) (*ConsClient, error) {
	c, err := smc.New(initBlp, id, cp)
	if err != nil {
		return nil, err
	}
	return &ConsClient{c}, nil
}

func (cc *ConsClient) Reconf(cp conf.Provider, prop *bp.Blueprint) (cnt int, err error) {
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(cc.Blueps[0]) == 1 {
		glog.V(3).Infof("C%d: Proposal is already in place.", cc.Id)
		return 0, nil
	}

	_, cnt, err = cc.Doreconf(cp, prop, 0, nil)
	return
}
