/*Package smclient implements the client side of both
the SmartMerge and Rambo algorithm.

The two algorithms use identical read and write methods.
But implement reconfiguration differently.

*/
package smclient

import (
	"context"
	"errors"

	"github.com/golang/glog"

	bp "github.com/relab/smartmerge/blueprints"
	conf "github.com/relab/smartmerge/confProvider"
	pb "github.com/relab/smartmerge/proto"
)

const Retry = 1
const MinSize = 3

// The smartmerge client. Stores a list of blueprints and the Id.
type SmClient struct {
	Blueps []*bp.Blueprint
	Id     uint32
}

func New(initBlp *bp.Blueprint, id uint32, cp conf.Provider) (*SmClient, error) {
	cnf := cp.FullC(initBlp)

	glog.Infof("New Client with Id: %d\n", id)

	_, err := cnf.SetCur(context.Background(), &pb.NewCur{Cur: initBlp, CurC: uint32(initBlp.Hash())})
	if err != nil {
		glog.Errorln("initial SetCur returned error: ", err)
		return nil, errors.New("Initial SetCur failed.")
	}
	return &SmClient{
		Blueps: []*bp.Blueprint{initBlp},
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
	if cnt > mcnt {
		return rs.Value, cnt
	}
	return rs.Value, mcnt
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

// Given a state returned from a regular read or Get, and a value to be written,
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

func (smc *SmClient) GetCur() *bp.Blueprint {
	return smc.Blueps[0].Copy()
}
