package consclient

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

type CRClient struct {
	*CClient
}

func NewCR(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*CRClient, error) {
	cc, err := New(initBlp, mgr, id)
	return &CRClient{cc}, err
}

//Atomic read
func (cr *CRClient) Read() (val []byte, cnt int) {
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
func (cr *CRClient) RRead() (val []byte, cnt int) {
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

func (cr *CRClient) Write(val []byte) int {
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
