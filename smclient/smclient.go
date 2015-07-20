package smclient

import (
	"github.com/relab/smartMerge/rpc"
	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
)

func majQuorum(bp *lat.Blueprint) int {
	return len(bp.Add)/2 +1
}

type SmClient struct {
	Blueps []*lat.Blueprint
	Confs []*rpc.Configuration
	mgr *rpc.Manager
	ID uint32
}

func NewSmClient(initBlp *lat.Blueprint, mgr *rpc.Manager, id uint32) (*SmClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Ids(), majQuorum(initBlp))
	if err != nil {
		return nil, err
	}
	return &SmClient{
		Blueps : []*lat.Blueprint{initBlp},
		Confs  : []*rpc.Configuration{conf},
		mgr    : mgr,
		ID     : id,
	       } , nil
}

//Atomic read
func (smc *SmClient) read() []byte {
	rs := smc.get()
	if rs == nil {
		return nil
	}
	smc.set(rs)
	return rs.Value
}

func (smc *SmClient) write(val []byte) {
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
