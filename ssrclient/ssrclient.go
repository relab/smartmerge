package ssrclient

import (
	"errors"

	"github.com/golang/glog"
	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
	smc "github.com/relab/smartMerge/smclient"
)

type SSRClient struct {
	*smc.SmClient
}

func New(initBlp *pb.Blueprint, id uint32, cp conf.Provider) (*SSRClient, error) {

	cnf := cp.FullC(initBlp)

	glog.Infof("New Client with Id: %d\n", id)

	_, err := cnf.SSetCur(&pb.NewCur{initBlp, uint32(initBlp.Len())})
	if err != nil {
		glog.Errorln("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}

	sc := &smc.SmClient{
		Blueps: []*pb.Blueprint{initBlp},
		Id:     id,
	}

	return &SSRClient{sc}, nil
}

//Atomic read
func (ssc *SSRClient) Read(cp conf.Provider) (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting Read")
	}
	var st *pb.State
	var err error

	st, cnt, err = ssc.Doreconf(cp, nil, false, nil)
	if err != nil {
		glog.Errorln("Error during Read", err)
		return nil, 0
	}

	if glog.V(3) {
		if cnt > 3 {
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
func (ssc *SSRClient) RRead(cp conf.Provider) (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting regular Read")
	}
	var st *pb.State
	var err error

	st, cnt, err = ssc.Doreconf(cp, nil, true, nil)

	if err != nil {
		glog.Errorln("Error during RRead")
		return nil, 0
	}
	if glog.V(3) {
		if cnt > 2 {
			glog.Infof("RRead used %d accesses\n", cnt)
		}
	}
	if st == nil {
		glog.Errorln("read returned nil state")
		return nil, cnt
	}
	return st.Value, cnt
}

func (ssc *SSRClient) Write(cp conf.Provider, val []byte) (cnt int) {
	if glog.V(5) {
		glog.Infoln("starting Write")
	}
	var err error

	_, cnt, err = ssc.Doreconf(cp, nil, false, val)

	if err != nil {
		glog.Errorln("Error during Write")
		return 0
	}
	if glog.V(3) {
		if cnt > 3 {
			glog.Infof("Write used %d accesses\n", cnt)
		}
	}
	return cnt
}

func (ssc *SSRClient) Reconf(cp conf.Provider, prop *pb.Blueprint) (cnt int, err error) {
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(ssc.Blueps[0]) == 1 {
		glog.V(3).Infof("C%d: Proposal is already in place.", ssc.Id)
		return 0, nil
	}

	_, cnt, err = ssc.Doreconf(cp, prop, true, nil)
	return
}
