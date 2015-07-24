package dynaclient

import (
	"errors"
	"fmt"

	lat "github.com/relab/smartMerge/directCombineLattice"
	//pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/rpc"
)

func majQuorum(bp *lat.Blueprint) int {
	return len(bp.Add)/2 + 1
}

type DynaClient struct {
	Blueps []*lat.Blueprint
	Confs  []*rpc.Configuration
	mgr    *rpc.Manager
	ID     uint32
}

func New(initBlp *lat.Blueprint, mgr *rpc.Manager, id uint32) (*DynaClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Ids(), majQuorum(initBlp))
	if err != nil {
		return nil, err
	}

	err = conf.SetCur(initBlp)
	if err != nil {
		fmt.Println("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &DynaClient{
		Blueps: []*lat.Blueprint{initBlp},
		Confs:  []*rpc.Configuration{conf},
		mgr:    mgr,
		ID:     id,
	}, nil
}

//Atomic read
func (dc *DynaClient) Read() (val []byte, cnt int) {
	val, cnt, err := dc.Traverse(nil, nil)
	if err != nil {
		fmt.Println("Traverse returned error: ", err)
	}
	return val, cnt
}

func (dc *DynaClient) Write(val []byte) int {
	_, cnt, err := dc.Traverse(nil, val)
	if err != nil {
		fmt.Println("Traverse returned error: ", err)
	}
	return cnt
}

func (dc *DynaClient) Reconf(bp *lat.Blueprint) (int, error) {
	_, cnt, err := dc.Traverse(bp, nil)
	return cnt, err
}

func (dc *DynaClient) GetCur() *lat.Blueprint {
	if len(dc.Blueps) == 0 {
		return nil
	}
	return dc.Blueps[0]
}
