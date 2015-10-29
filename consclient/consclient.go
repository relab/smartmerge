package consclient

import (
	"errors"
	"time"

	"github.com/golang/glog"

	pb "github.com/relab/smartMerge/proto"
)

func majQuorum(bp *pb.Blueprint) int {
	return len(bp.Add)/2 + 1
}

var ConfTimeout = 1 * time.Second

type CClient struct {
	Blueps []*pb.Blueprint
	Confs  []*pb.Configuration
	mgr    *pb.Manager
	ID     uint32
}

func New(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*CClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Add, majQuorum(initBlp), ConfTimeout)
	if err != nil {
		return nil, err
	}

	glog.Infof("New Client with Id: %d\n", id)

	_, err = conf.CSetState(&pb.CNewCur{Cur: initBlp, CurC: uint32(initBlp.Len())})
	if err != nil {
		glog.Errorln("initial SetCur returned error: ", err)
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
	if glog.V(5) {
		glog.Infoln("starting Read")
	}

	rs, cnt := cc.get()
	if rs == nil {
		return nil, cnt
	}

	mcnt := cc.set(rs)

	if glog.V(3) {
		if cnt > 1 {
			glog.Infof("get used %d accesses\n", cnt)
		}
		if mcnt > 1 {
			glog.Infof("set used %d accesses\n", mcnt)
		}
	}
	return rs.Value, cnt + mcnt
}

//Regular read
func (cc *CClient) RRead() (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting regular Read")
	}
	rs, cnt := cc.get()
	if rs == nil {
		return nil, cnt
	}
	if glog.V(3) {
		if cnt > 1 {
			glog.Infof("get used %d accesses\n", cnt)
		}
	}
	return rs.Value, cnt
}

func (cc *CClient) Write(val []byte) int {
	if glog.V(5) {
		glog.Infoln("starting Write")
	}
	rs, cnt := cc.get()
	if rs == nil {
		rs = &pb.State{Value: val, Timestamp: 1, Writer: cc.ID}
	} else {
		rs.Value = val
		rs.Timestamp++
		rs.Writer = cc.ID
	}
	mcnt := cc.set(rs)
	if glog.V(3) {
		if cnt > 1 {
			glog.Infof("get used %d accesses\n", cnt)
		}
		if mcnt > 1 {
			glog.Infof("set used %d accesses\n", mcnt)
		}
	}
	return cnt + mcnt
}

func (cc *CClient) GetCur() *pb.Blueprint {
	if len(cc.Blueps) == 0 {
		return nil
	}
	return cc.Blueps[0]
}
