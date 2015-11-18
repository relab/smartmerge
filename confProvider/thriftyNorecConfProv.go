package confProvider

import (
	"time"

	"github.com/golang/glog"

	pb "github.com/relab/smartMerge/proto"
)

var ConfTimeout = 1 * time.Second
var TryTimeout = 10 * time.Millisecond

type Provider interface {
	FullC(*pb.Blueprint) *pb.Configuration
	ReadC(*pb.Blueprint, []int) *pb.Configuration
	WriteC(*pb.Blueprint, []int) *pb.Configuration
}

type ThriftyNorecConfP struct {
	mgr *pb.Manager
	id  int
}

func NewProvider(mgr *pb.Manager, id int) *ThriftyNorecConfP {
	return &ThriftyNorecConfP{mgr, id}
}

func (cp *ThriftyNorecConfP) chooseQ(ids []int, q int) (quorum []int) {
	if q > len(ids) {
		glog.Fatalf("Trying to choose %d nodes, out of %d\n", q, len(ids))
	}

	quorum = make([]int, q)
	start := cp.id % len(ids)
	if start+q <= len(ids) {
		copy(quorum, ids[start:])
		return quorum
	}
	copy(quorum, ids[start:])
	copy(quorum[len(ids)-start:], ids)
	return quorum
}

func (cp *ThriftyNorecConfP) ReadC(blp *pb.Blueprint, rids []int) *pb.Configuration {
	cids := cp.mgr.ToIds(blp.Ids())
	rq := len(cids) - blp.Quorum() + 1 //read quorum
	newcids := pb.Difference(cids, rids)

	if len(cids)-len(newcids) >= rq {
		//We already have enough replies.
		return nil
	}

	// I already have y := len(cids) - len(newcids) many replies.
	// I still need rq - y many.
	newcids = cp.chooseQ(newcids, rq-len(cids)+len(newcids))

	// With quorum size 1, a read quorum contains all processes.
	cnf, err := cp.mgr.NewConfiguration(newcids, 1, TryTimeout)
	if err != nil {
		glog.Fatalln("could not get read config")
	}

	return cnf
}

func (cp *ThriftyNorecConfP) WriteC(blp *pb.Blueprint, rids []int) *pb.Configuration {
	cids := cp.mgr.ToIds(blp.Ids())
	q := blp.Quorum()
	newcids := pb.Difference(cids, rids)

	if len(cids)-len(newcids) >= q {
		//We already have enough replies.
		return nil
	}

	// I already have y := len(cids) - len(newcids) many replies.
	// I still need q - y many.
	newcids = cp.chooseQ(newcids, q-len(cids)+len(newcids))
	cnf, err := cp.mgr.NewConfiguration(newcids, len(newcids), TryTimeout)
	if err != nil {
		glog.Fatalln("could not get read config")
	}

	return cnf
}

func (cp *ThriftyNorecConfP) FullC(blp *pb.Blueprint) *pb.Configuration {
	cids := cp.mgr.ToIds(blp.Ids())
	q := blp.Quorum()

	cnf, err := cp.mgr.NewConfiguration(cids, q, ConfTimeout)
	if err != nil {
		glog.Fatalln("could not get config")
	}

	return cnf
}
