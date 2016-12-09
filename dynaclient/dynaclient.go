package dynaclient

import (
	"errors"

	"github.com/golang/glog"

	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
)

type DynaClient struct {
	Blueps []*pb.Blueprint
	Confs  []*pb.Configuration
	ID     uint32
}

func New(initBlp *pb.Blueprint, id uint32, cp conf.Provider) (*DynaClient, error) {
	conf := cp.FullC(initBlp)

	glog.Infof("New Client with Id: %d\n", id)

	_, err := conf.DSetCur(&pb.NewCur{initBlp, uint32(initBlp.Len())})
	if err != nil {
		glog.Errorln("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &DynaClient{
		Blueps: []*pb.Blueprint{initBlp},
		Confs:  []*pb.Configuration{conf},
		ID:     id,
	}, nil
}

//Atomic read
func (dc *DynaClient) Read(cp conf.Provider) (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting Read")
	}
	val, cnt, err := dc.Traverse(cp, nil, nil, false)
	if err != nil {
		glog.Infoln("Traverse returned error: ", err)
	}
	if glog.V(3) {
		if cnt > 2 {
			glog.Infof("read used %d accesses\n", cnt)
		}
	}
	return val, cnt
}

//Regular read
func (dc *DynaClient) RRead(cp conf.Provider) (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting regular Read")
	}

	val, cnt, err := dc.Traverse(cp, nil, nil, true)
	if err != nil {
		glog.Infoln("Traverse returned error: ", err)
	}
	if glog.V(3) {
		if cnt > 1 {
			glog.Infof("regular read used %d accesses\n", cnt)
		}
	}
	return val, cnt
}

func (dc *DynaClient) Write(cp conf.Provider, val []byte) int {
	if glog.V(5) {
		glog.Infoln("starting write")
	}
	_, cnt, err := dc.Traverse(cp, nil, val, false)
	if err != nil {
		glog.Infoln("Traverse returned error: ", err)
	}
	if glog.V(3) {
		if cnt > 2 {
			glog.Infof("write used %d accesses\n", cnt)
		}
	}
	return cnt
}

func (dc *DynaClient) Reconf(cp conf.Provider, bp *pb.Blueprint) (int, error) {
	if glog.V(3) {
		glog.Infoln("starting reconf")
	}

	_, cnt, err := dc.Traverse(cp, bp, nil, false)
	if glog.V(3) {
		glog.Infof("reconf used %d accesses\n", cnt)
	}
	return cnt, err
}

func (dc *DynaClient) GetCur(cp conf.Provider) *pb.Blueprint {
	return dc.Blueps[len(dc.Blueps)-1].Copy()
}
