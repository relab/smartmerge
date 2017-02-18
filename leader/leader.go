package leader

import (
	"github.com/golang/glog"
	bp "github.com/relab/smartMerge/blueprints"
	conf "github.com/relab/smartMerge/confProvider"
	cs "github.com/relab/smartMerge/consclient"
)

type Leader struct {
	*cs.ConsClient
	propC    chan *bp.Blueprint
	getdoneC chan chan struct{}
	stopC    chan bool
	cp       conf.Provider
}

func New(initBlp *bp.Blueprint, id uint32, cp conf.Provider) (*Leader, error) {
	cc, err := cs.New(initBlp, id, cp)
	if err != nil {
		return nil, err
	}
	return &Leader{
		ConsClient: cc,
		propC:      make(chan *bp.Blueprint, 0),
		getdoneC:   make(chan chan struct{}, 0),
		stopC:      make(chan bool, 0),
		cp:         cp,
	}, nil
}

func (l *Leader) Propose(prop *bp.Blueprint) {
	l.propC <- prop
	doneC := <-l.getdoneC
	<-doneC
}

func (l *Leader) Stop() {
	l.stopC <- true
}

func (l *Leader) Run() {
	go l.run()
}

func (l *Leader) run() {
run_for:
	for {
		doneC := make(chan struct{})
		var prop *bp.Blueprint
		select {
		case <-l.stopC:
			break run_for
		case prop = <-l.propC:
			l.getdoneC <- doneC
		}
		for more := true; more; {
			select {
			case <-l.stopC:
				break run_for
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
