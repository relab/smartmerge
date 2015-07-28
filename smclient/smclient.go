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

	fmt.Println("Start initial setcur")
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
func (smc *SmClient) Read() (val []byte, cnt int) {
	rs, cnt := smc.get()
	if rs == nil {
		return nil, cnt
	}

	mcnt := smc.set(rs)
	return rs.Value, cnt + mcnt
}

func (smc *SmClient) Write(val []byte) int {
	rs, cnt := smc.get()
	if rs == nil {
		rs = &pb.State{Value: val, Timestamp: 1, Writer: smc.ID}
	} else {
		rs.Value = val
		rs.Timestamp++
		rs.Writer = smc.ID
	}
	mcnt := smc.set(rs)
	return cnt + mcnt
}

func (smc *SmClient) GetCur() *lat.Blueprint {
	if len(smc.Blueps) == 0 {
		return nil
	}
	return smc.Blueps[0]
}
