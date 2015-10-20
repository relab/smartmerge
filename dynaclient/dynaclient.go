package dynaclient

import (
	"errors"
	"fmt"
	"time"

	pb "github.com/relab/smartMerge/proto"
)

func majQuorum(bp *pb.Blueprint) int {
	return len(bp.Add)/2 + 1
}

type DynaClient struct {
	Blueps []*pb.Blueprint
	Confs  []*pb.Configuration
	mgr    *pb.Manager
	ID     uint32
}

func New(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*DynaClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Add, majQuorum(initBlp), 2* time.Second)
	if err != nil {
		return nil, err
	}

	_, err = conf.DSetCur(&pb.NewCur{initBlp, uint32(initBlp.Len())})
	if err != nil {
		fmt.Println("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &DynaClient{
		Blueps: []*pb.Blueprint{initBlp},
		Confs:  []*pb.Configuration{conf},
		mgr:    mgr,
		ID:     id,
	}, nil
}

//Atomic read
func (dc *DynaClient) Read() (val []byte, cnt int) {
	val, cnt, err := dc.Traverse(nil, nil, false)
	if err != nil {
		fmt.Println("Traverse returned error: ", err)
	}
	return val, cnt
}

//Regular read
func (dc *DynaClient) RRead() (val []byte, cnt int) {
	val, cnt, err := dc.Traverse(nil, nil, true)
	if err != nil {
		fmt.Println("Traverse returned error: ", err)
	}
	return val, cnt
}

func (dc *DynaClient) Write(val []byte) int {
	_, cnt, err := dc.Traverse(nil, val, false)
	if err != nil {
		fmt.Println("Traverse returned error: ", err)
	}
	return cnt
}

func (dc *DynaClient) Reconf(bp *pb.Blueprint) (int, error) {
	_, cnt, err := dc.Traverse(bp, nil, false)
	return cnt, err
}

func (dc *DynaClient) GetCur() *pb.Blueprint {
	if len(dc.Blueps) == 0 {
		return nil
	}
	return dc.Blueps[0]
}
