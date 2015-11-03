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

type COptClient struct {
	*CClient
}

func NewOpt(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*COptClient, error) {
	cr, err := New(initBlp, mgr, id)
	return &COptClient{cr}, err
}

func (cc *COptClient) Read() (val []byte, cnt int) {
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
func (cc *COptClient) RRead() (val []byte, cnt int) {
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

func (cc *COptClient) Write(val []byte) int {
	if glog.V(5) {
		glog.Infoln("starting Write")
	}
	rs, cnt := cc.get()
	if rs == nil && cnt == 0 {
		return 0
	}
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
