package smclient

import (
	"errors"
	"time"

	"github.com/golang/glog"

	pb "github.com/relab/smartMerge/proto"
)

var ConfTimeout = 1 * time.Second
var TryTimeout = 10 * time.Millisecond
var MinSize = 3

type SmClient struct {
	Blueps []*pb.Blueprint
	Confs  []*pb.Configuration
	curRead *pb.Configuration
	curWrite *pb.Configuration	
	mgr    *pb.Manager
	ID     uint32
	doCons bool
}

func New(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32,cons bool) (*SmClient, error) {
	conf, err := mgr.NewConfiguration(initBlp.Ids(), initBlp.Quorum(), ConfTimeout)
	if err != nil {
		return nil, err
	}

	glog.Infof("New Client with Id: %d\n", id)

	_, err = conf.SetCur(&pb.NewCur{initBlp, uint32(initBlp.Len())})
	if err != nil {
		glog.Errorln("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &SmClient{
		Blueps: []*pb.Blueprint{initBlp},
		Confs:  []*pb.Configuration{conf},
		mgr:    mgr,
		ID:     id,
		doCons: cons,
	}, nil
}

//Atomic read
func (smc *SmClient) Read() (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting Read")
	}
	rs, cnt := smc.get()
	if rs == nil {
		return nil, cnt
	}

	mcnt := smc.set(rs)

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
func (smc *SmClient) RRead() (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting regular Read")
	}
	rs, cnt := smc.get()
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

func (smc *SmClient) Write(val []byte) int {
	if glog.V(5) {
		glog.Infoln("starting Write")
	}
	rs, cnt := smc.get()
	if rs == nil && cnt == 0 {
		return 0
	}
	if rs == nil {
		rs = &pb.State{Value: val, Timestamp: 1, Writer: smc.ID}
	} else {
		rs.Value = val
		rs.Timestamp++
		rs.Writer = smc.ID
	}
	mcnt := smc.set(rs)
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

// We should probably only return a copy.
func (smc *SmClient) GetCur() *pb.Blueprint {
	smc.set(nil)
	b := smc.Blueps[0].Copy()
	return b
}
