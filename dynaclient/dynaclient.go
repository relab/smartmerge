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
func (dc *DynaClient) Read() []byte {
	val, err := dc.Traverse(nil, nil)
	if err != nil {
		fmt.Println("Traverse returned error: ", err)
	}
	return val
}

func (dc *DynaClient) Write(val []byte) {
	_, err := dc.Traverse(nil, val)
	if err != nil {
		fmt.Println("Traverse returned error: ", err)
	}
	return
}

func (dc *DynaClient) Reconf(bp *lat.Blueprint) {
	_, err := dc.Traverse(bp, nil)
	if err != nil {
		fmt.Println("Traverse returned error: ", err)
	}
	return
}
