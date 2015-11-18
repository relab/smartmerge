package smclient

import (
	"errors"

	"github.com/golang/glog"

	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
)

const Retry = 1
const MinSize = 3

type SmClient struct {
	Blueps   []*pb.Blueprint
	Id		 uint32
}

func New(initBlp *pb.Blueprint, id uint32, cp conf.Provider) (*SmClient, error) {
	cnf := cp.FullC(initBlp)

	glog.Infof("New Client with Id: %d\n", id)

	_, err := cnf.SetCur(&pb.NewCur{initBlp, uint32(initBlp.Len())})
	if err != nil {
		glog.Errorln("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &SmClient{
		Blueps: []*pb.Blueprint{initBlp},
		Id:     id,
	}, nil
}

//Atomic read
func (smc *SmClient) Read(cp conf.Provider) (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting Read")
	}
	rs, cnt := smc.get(cp)
	if rs == nil {
		return nil, cnt
	}

	mcnt := smc.set(cp, rs)

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
func (smc *SmClient) RRead(cp conf.Provider) (val []byte, cnt int) {
	if glog.V(5) {
		glog.Infoln("starting regular Read")
	}
	rs, cnt := smc.get(cp)
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

func (smc *SmClient) Write(cp conf.Provider, val []byte) int {
	if glog.V(5) {
		glog.Infoln("starting Write")
	}
	rs, cnt := smc.get(cp)
	if rs == nil && cnt == 0 {
		return 0
	}
	rs = smc.WriteValue(&val, rs)
	
	mcnt := smc.set(cp, rs)
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

// Given a state returned from a regular read, and a value to be written,
// getWriteValue finds the correct state to write. 
// The value is passed by pointer, and set to nil, to avoid reseting the write value.
func (smc *SmClient) WriteValue(val *[]byte, st *pb.State) *pb.State {
	if val == nil || *val == nil {
		return st
	}
	if st == nil {
		return &pb.State{Value: *val, Timestamp: 1, Writer: smc.Id}
	}
	st = &pb.State{Value: *val, Timestamp: st.Timestamp + 1, Writer: smc.Id}
	*val = nil
	return st
}

func (smc *SmClient) GetCur(cp conf.Provider) *pb.Blueprint {
	smc.set(cp, nil)
	return smc.Blueps[0].Copy()
}

