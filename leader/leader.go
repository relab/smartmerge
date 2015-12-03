package leader

import (
	"github.com/golang/glog"
	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
	sm "github.com/relab/smartMerge/smclient"
)

type Leader struct {
	*sm.SmClient
	propC    chan *pb.Blueprint
	getdoneC chan chan struct{}
	cp       conf.Provider
}

func New(initBlp *pb.Blueprint, id uint32, cp conf.Provider) (*Leader, error) {
	smc, err := sm.New(initBlp, id, cp)
	if err != nil {
		return nil, err
	}
	return &Leader{
		SmClient: smc,
		propC:    make(chan *pb.Blueprint, 0),
		getdoneC: make(chan chan struct{}, 0),
		cp:       cp,
	}, nil
}

func (l *Leader) propose(prop *pb.Blueprint) {
	l.propC <- prop
	doneC := <-l.getdoneC
	<-doneC
}

func (l *Leader) run() {
	for {
		doneC := make(chan struct{})
		prop := <-l.propC
		l.getdoneC <- doneC
		for more := true; more; {
			select {
			case x := <-l.propC:
				prop = prop.Merge(x)
				l.getdoneC <- doneC
			default:
				more = false
			}
		}

		//Should we add a check, whether the proposal is actually holding anything new?
		_, err := l.Reconf(l.cp, prop)
		if err != nil {
			glog.Errorln("Reconf returned error:", err)
		}
		if glog.V(3) {
			glog.Infoln("Reconfiguration returned.")
		}
		close(doneC)
	}
}
