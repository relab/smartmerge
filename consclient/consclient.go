package consclient

import (
	"errors"
	"fmt"
	"time"

	pb "github.com/relab/smartMerge/proto"
)

func majQuorum(bp *pb.Blueprint) int {
	return len(bp.Add)/2 + 1
}

type CClient struct {
	Blueps []*pb.Blueprint
	Confs  []*pb.Configuration
	mgr    *pb.Manager
	ID     uint32
}

func New(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*CClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Add, majQuorum(initBlp), 2* time.Second)
	if err != nil {
		return nil, err
	}

	_, err = conf.CSetState(&pb.CNewCur{Cur: initBlp, CurC: uint32(initBlp.Len())})
	if err != nil {
		fmt.Println("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &CClient{
		Blueps: []*pb.Blueprint{initBlp},
		Confs:  []*pb.Configuration{conf},
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

//Regular read
func (cc *CClient) RRead() (val []byte, cnt int) {
	rs, cnt := cc.get()
	if rs == nil {
		return nil, cnt
	}
	return rs.Value, cnt
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

func (cc *CClient) GetCur() *pb.Blueprint {
	if len(cc.Blueps) == 0 {
		return nil
	}
	return cc.Blueps[0]
}
