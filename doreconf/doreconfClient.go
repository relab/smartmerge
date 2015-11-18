package doreconf

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
	conf "github.com/relab/smartMerge/confProvider"
	smc "github.com/relab/smartMerge/smclient"
	cc "github.com/relab/smartMerge/consclient"
)

type Reconfer interface {
	Doreconf(conf.Provider, *pb.Blueprint, bool, []byte) (*pb.State, int, error)
	Reconf(conf.Provider, *pb.Blueprint) (int, error) 
	GetCur(conf.Provider) *pb.Blueprint
}

type DoreconfClient struct {
	Reconfer
}

func NewSM(initBlp *pb.Blueprint, id uint32, cp conf.Provider) (*DoreconfClient, error) {

	rec, err := smc.New(initBlp, id, cp)
	
	if err != nil {
		return nil, err
	}
	
	return &DoreconfClient{rec}, nil
}

func NewCons(initBlp *pb.Blueprint, id uint32, cp conf.Provider) (*DoreconfClient, error) {

	rec, err := cc.New(initBlp, id, cp)
	
	if err != nil {
		return nil, err
	}
	
	return &DoreconfClient{rec}, nil
}


//Atomic read
func (drc *DoreconfClient) Read(cp conf.Provider) (val []byte, cnt int) {
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
func (drc *DoreconfClient) RRead(cp conf.Provider) (val []byte, cnt int) {
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

func (drc *DoreconfClient) Write(cp conf.Provider, val []byte) (cnt int) {
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