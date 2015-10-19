package smclient

import (
	"errors"
	"fmt"
	"time"

	pb "github.com/relab/smartMerge/proto"
)

func majQuorum(bp *pb.Blueprint) int {
	return len(bp.Add)/2 + 1
}

type SmClient struct {
	Blueps []*pb.Blueprint
	Confs  []*pb.Configuration
	mgr    *pb.Manager
	ID     uint32
}

func New(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*SmClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Add, majQuorum(initBlp), 2*time.Second)
	if err != nil {
		return nil, err
	}

	_,err = conf.SetCur(&pb.NewCur{initBlp, uint32(initBlp.Len())})
	if err != nil {
		fmt.Println("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &SmClient{
		Blueps: []*pb.Blueprint{initBlp},
		Confs:  []*pb.Configuration{conf},
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

//Regular read
func (smc *SmClient) RRead() (val []byte, cnt int) {
	rs, cnt := smc.get()
	if rs == nil {
		return nil, cnt
	}
	return rs.Value, cnt
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

func (smc *SmClient) GetCur() *pb.Blueprint {
	if len(smc.Blueps) == 0 {
		return nil
	}
	return smc.Blueps[0]
}
