package confProvider

import (
	"time"

	"github.com/golang/glog"
	bp "github.com/relab/smartmerge/blueprints"
	pb "github.com/relab/smartmerge/proto"
	qspec "github.com/relab/smartmerge/qfuncs"
)

var ConfTimeout = 1 * time.Second
var TryTimeout = 500 * time.Millisecond

type Provider interface {
	FullC(*bp.Blueprint) *pb.Configuration
	ReadC(*bp.Blueprint, []uint32) *pb.Configuration
	WriteC(*bp.Blueprint, []uint32) *pb.Configuration
	SingleC(*bp.Blueprint) *pb.Configuration
	WriteCNoS(*bp.Blueprint, []uint32) *pb.Configuration
}

type ThriftyNorecConfP struct {
	mgr *pb.Manager
	id  int
}

func NewProvider(mgr *pb.Manager, id int) *ThriftyNorecConfP {
	return &ThriftyNorecConfP{mgr, id}
}

func (cp *ThriftyNorecConfP) chooseQ(ids []uint32, q int) (quorum []uint32) {
	if q > len(ids) {
		glog.Fatalf("Trying to choose %d nodes, out of %d\n", q, len(ids))
	}

	quorum = make([]uint32, q)
	start := cp.id % len(ids)
	if start+q <= len(ids) {
		copy(quorum, ids[start:])
		return quorum
	}
	copy(quorum, ids[start:])
	copy(quorum[len(ids)-start:], ids)
	return quorum
}

func (cp *ThriftyNorecConfP) ReadC(blp *bp.Blueprint, rids []uint32) *pb.Configuration {
	newcids, qs := cp.readC(blp, rids)
	if newcids == nil {
		return nil
	}
	cnf, err := cp.mgr.NewConfiguration(newcids, qs)
	if err != nil {
		glog.Fatalln("could not get read config")
	}

	return cnf
}

// readC is an easily testable version of ReadC
func (cp *ThriftyNorecConfP) readC(blp *bp.Blueprint, rids []uint32) (newcids []uint32, qs *qspec.SMQuorumSpec) {
	cids := blp.Ids()
	rq := qspec.ReadQuorum(blp.Quorum(), len(cids))
	newcids = bp.Difference(cids, rids) //Nodes in the configuration (cids), that have not yet replies (not in rids)

	if len(cids)-len(newcids) >= rq {
		//We already have enough replies.
		return nil, nil
	}

	// I already have y := len(cids) - len(newcids) many replies.
	// I still need rq - y many.
	newcids = cp.chooseQ(newcids, rq-(len(cids)-len(newcids)))

	// With quorum size 1, a read quorum contains all processes.
	qs = qspec.NewSMQSpec(1, len(newcids))

	return newcids, qs
}

func (cp *ThriftyNorecConfP) WriteC(blp *bp.Blueprint, rids []uint32) *pb.Configuration {
	cids := blp.Ids()
	q := qspec.WriteQuorum(blp.Quorum(), len(cids))
	newcids := bp.Difference(cids, rids)

	if len(cids)-len(newcids) >= q {
		//We already have enough replies.
		return nil
	}

	// I already have y := len(cids) - len(newcids) many replies.
	// I still need q - y many.
	newcids = cp.chooseQ(newcids, q-len(cids)+len(newcids))
	qs := qspec.NewSMQSpec(len(newcids), len(newcids))

	cnf, err := cp.mgr.NewConfiguration(newcids, qs)
	if err != nil {
		glog.Fatalln("could not get read config")
	}

	return cnf
}

func (cp *ThriftyNorecConfP) FullC(blp *bp.Blueprint) *pb.Configuration {
	cids := blp.Ids()

	qs := qspec.SMQSpecFromBP(blp)
	cnf, err := cp.mgr.NewConfiguration(cids, qs)
	if err != nil {
		glog.Fatalln("could not get config")
	}

	return cnf
}

func (cp *ThriftyNorecConfP) SingleC(blp *bp.Blueprint) *pb.Configuration {
	cids := blp.Ids()
	m := cids[0]
	for _, id := range cids {
		if m < id {
			m = id
		}
	}
	cids = []uint32{m}

	qs := qspec.NewSMQSpec(1, 1)
	cnf, err := cp.mgr.NewConfiguration(cids, qs)
	if err != nil {
		glog.Fatalln("could not get config")
	}

	return cnf
}

func (cp *ThriftyNorecConfP) WriteCNoS(blp *bp.Blueprint, rids []uint32) *pb.Configuration {
	cids := blp.Ids()
	m := cids[0]
	for _, id := range cids {
		if m < id {
			m = id
		}
	}

	q := blp.Quorum()
	newcids := bp.Difference(cids, rids)

	if len(cids)-len(newcids) >= q {
		//We already have enough replies.
		return nil
	}

	// I already have y := len(cids) - len(newcids) many replies.
	// I still need q - y many.
	y := len(cids) - len(newcids)
	newcids = bp.Difference(newcids, []uint32{m})
	newcids = cp.chooseQ(newcids, q-y)
	qs := qspec.NewSMQSpec(len(newcids), len(newcids))
	cnf, err := cp.mgr.NewConfiguration(newcids, qs)
	if err != nil {
		glog.Fatalln("could not get read config")
	}

	return cnf
}
