package smclient

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

type SmRClient struct {
	*SmClient
}

func NewSmR(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*SmRClient, error) {
	cr, err := New(initBlp, mgr, id)
	return &SmRClient{cr}, err
}

//Atomic read
func (cr *SmRClient) Read() (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting Read")
	}
	st, cnt, err := cr.reconf(nil, false, nil)
	if err != nil {
		glog.Errorln("Error during Read", err)
		return nil, 0
	}

	if glog.V(3) {
		if cnt > 2 {
			glog.Infof("Read used %d accesses\n", cnt)
		}
	}
	if st == nil {
		glog.Errorln("read returned nil state")
		return nil, cnt
	}
	return st.Value, cnt
}

//Regular read
func (cr *SmRClient) RRead() (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting regular Read")
	}
	st, cnt, err := cr.reconf(nil, true, nil)
	if err != nil {
		glog.Errorln("Error during RRead")
		return nil, 0
	}
	if glog.V(3) {
		if cnt > 1 {
			glog.Infof("RRead used %d accesses\n", cnt)
		}
	}
	if st == nil {
		glog.Errorln("read returned nil state")
		return nil, cnt
	}
	return st.Value, cnt
}

func (cr *SmRClient) Write(val []byte) int {
	if glog.V(5) {
		glog.Infoln("starting Write")
	}
	_, cnt, err := cr.reconf(nil, false, val)
	if err != nil {
		glog.Errorln("Error during Write")
		return 0
	}
	if glog.V(3) {
		if cnt > 2 {
			glog.Infof("Write used %d accesses\n", cnt)
		}
	}
	return cnt
}

type SmOptClient struct {
	*SmClient
}

func NewOpt(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*SmOptClient, error) {
	cr, err := New(initBlp, mgr, id)
	return &SmOptClient{cr}, err
}

func (smc *SmOptClient) Read() (val []byte, cnt int) {
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
func (smc *SmOptClient) RRead() (val []byte, cnt int) {
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

func (smc *SmOptClient) Write(val []byte) int {
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
