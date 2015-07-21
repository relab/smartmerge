package smclient

import (
	"errors"
	"fmt"
	
	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/rpc"
)

func majQuorum(bp *lat.Blueprint) int {
	return len(bp.Add)/2 + 1
}

type SmClient struct {
	Blueps []*lat.Blueprint
	Confs  []*rpc.Configuration
	mgr    *rpc.Manager
	ID     uint32
}

func New(initBlp *lat.Blueprint, mgr *rpc.Manager, id uint32) (*SmClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Ids(), majQuorum(initBlp))
	if err != nil {
		return nil, err
	}
	
	err = conf.SetCur(initBlp)
	if err != nil {
		fmt.Println("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &SmClient{
		Blueps: []*lat.Blueprint{initBlp},
		Confs:  []*rpc.Configuration{conf},
		mgr:    mgr,
		ID:     id,
	}, nil
}

//Atomic read
func (smc *SmClient) Read() []byte {
	rs := smc.get()
	if rs == nil {
		return nil
	}

	smc.set(rs)
	return rs.Value
}

func (smc *SmClient) Write(val []byte) {
	rs := smc.get()
	if rs == nil {
		rs = &pb.State{Value: val, Timestamp: 1, Writer: smc.ID}
	} else {
		rs.Value = val
		rs.Timestamp++
		rs.Writer = smc.ID
	}
	smc.set(rs)
}
