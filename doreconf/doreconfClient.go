package doreconf

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
	confP "github.com/relab/smartMerge/confProvider"
	smc "github.com/relab/smartMerge/smclient"
	cc "github.com/relab/smartMerge/consclient"
)

type Reconfer interface {
	Doreconf(confP.ConfigProvider, *pb.Blueprint, bool, []byte) (*pb.State, int, error)
	Reconf(confP.ConfigProvider, *pb.Blueprint) (int, error) 
	GetCur(confP.ConfigProvider) *pb.Blueprint
}

type DoreconfClient struct {
	Reconfer
}

func New(initBlp *pb.Blueprint, id uint32, cp confP.ConfigProvider, cons bool) (*DoreconfClient, error) {
	var rec Reconfer
	var err error
	
	if cons {
		rec, err = cc.New(initBlp, id, cp)
	} else {
		rec, err = smc.New(initBlp, id, cp)
	}
	
	if err != nil {
		return nil, err
	}
	
	return &DoreconfClient{rec}, nil
}

//Atomic read
func (drc *DoreconfClient) Read(cp confP.ConfigProvider) (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting Read")
	}
	var st *pb.State
	var err error

	st, cnt, err = drc.Doreconf(cp, nil, false, nil)
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
func (drc *DoreconfClient) RRead(cp confP.ConfigProvider) (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting regular Read")
	}
	var st *pb.State
	var err error

	st, cnt, err = drc.Doreconf(cp, nil, true, nil)

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

func (drc *DoreconfClient) Write(cp confP.ConfigProvider, val []byte) (cnt int) {
	if glog.V(5) {
		glog.Infoln("starting Write")
	}
	var err error

	_, cnt, err = drc.Doreconf(cp, nil, false, val)

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