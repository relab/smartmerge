package consclient

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

type CClient struct {
	Blueps []*lat.Blueprint
	Confs  []*rpc.Configuration
	mgr    *rpc.Manager
	ID     uint32
}

func New(initBlp *lat.Blueprint, mgr *rpc.Manager, id uint32) (*CClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Ids(), majQuorum(initBlp))
	if err != nil {
		return nil, err
	}

	err = conf.CSetState(initBlp, nil)
	if err != nil {
		fmt.Println("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &CClient{
		Blueps: []*lat.Blueprint{initBlp},
		Confs:  []*rpc.Configuration{conf},
		mgr:    mgr,
		ID:     id,
	}, nil
}

//Atomic read
func (cc *CClient) Read() (val []byte, cnt int) {
	rs, cnt := cc.get()
	if rs == nil {
		return nil, cnt
	}

	mcnt := cc.set(rs)
	return rs.Value, cnt + mcnt
}

func (cc *CClient) Write(val []byte) int {
	rs, cnt := cc.get()
	if rs == nil {
		rs = &pb.State{Value: val, Timestamp: 1, Writer: cc.ID}
	} else {
		rs.Value = val
		rs.Timestamp++
		rs.Writer = cc.ID
	}
	mcnt := cc.set(rs)
	return cnt + mcnt
}

func (cc *CClient) GetCur() *lat.Blueprint {
	if len(cc.Blueps) == 0 {
		return nil
	}
	return cc.Blueps[0]
}
