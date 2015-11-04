package consclient

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (cc *COptClient) get() (rs *pb.State, cnt int) {
	cnt = 0
	cur := 0
	rid := make([]uint32, 0)
	for i := 0; i < len(cc.Blueps); i++ {
		if i < cur {
			continue
		}
		cnt++
		var cnf *pb.Configuration

		if i > 0 {
			cnf = cc.createX(rid, cc.Blueps[i].Ids(), false)
			if cnf == nil {
				glog.V(4).Infoln("We can skip contacting next configuration.")
				continue
			}
		} else {
			cnf = cc.Confs[0]
		}

		read, err := cnf.CReadS(&pb.Conf{uint32(cc.Blueps[i].Len()), uint32(cc.Blueps[cur].Len())})

		if err != nil && (read == nil || read.Reply == nil) {
			glog.Errorln("error from AReadS: ", err)
			//No Quorum Available. Retry
			return nil, 0
		}
		if glog.V(6) {
			glog.Infoln("AReadS returned with replies from ", read.MachineIDs)
		}

		cur = cc.handleNewCur(cur, read.Reply.GetCur(), false)

		cc.handleNext(i, read.Reply.GetNext(), false)

		if rs.Compare(read.Reply.GetState()) == 1 {
			rs = read.Reply.GetState()
		}

		if len(cc.Blueps) > i+1 {
			rid = pb.Union(rid, read.MachineIDs)
		}

	}
	if cur > 0 {
		cc.Confs[0] = cc.create(cc.Blueps[cur])
		cc.Blueps = cc.Blueps[cur:]
	}
	return
}

func (cc *COptClient) set(rs *pb.State) int {
	cnt := 0
	cur := 0
	rid := make([]uint32, 0)
	for i := 0; i < len(cc.Blueps); i++ {
		if i < cur {
			continue
		}
		cnt++
		var cnf *pb.Configuration

		if i > 0 {
			cnf = cc.createX(rid, cc.Blueps[i].Ids(), true)
			if cnf == nil {
				glog.V(4).Infoln("We can skip contacting next configuration.")
				continue
			}
		} else {
			cnf = cc.Confs[0]
		}

		write, err := cnf.CWriteS(&pb.WriteS{rs, &pb.Conf{uint32(cc.Blueps[i].Len()), uint32(cc.Blueps[cur].Len())}})

		if err != nil {
			glog.Errorln("AWriteS returned error, ", err)
			return 0
		}
		if glog.V(6) {
			glog.Infoln("AWriteS returned, with replies from ", write.MachineIDs)
		}

		cur = cc.handleNewCur(cur, write.Reply.GetCur(), false)
		cc.handleNext(i, write.Reply.GetNext(), false)

		if len(cc.Blueps) > i+1 {
			rid = pb.Union(rid, write.MachineIDs)
		}
	}

	if cur > 0 {
		cc.Confs[0] = cc.create(cc.Blueps[cur])
		cc.Blueps = cc.Blueps[cur:]
	}

	return cnt
}

func (cc *COptClient) handleNewCur(cur int, newCur *pb.ConfReply, createconf bool) int {
	if newCur == nil {
		return cur
	}
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Cur.Len(), cc.Blueps[cur].Len())
	}
	return cc.findorinsert(cur, newCur.Cur, createconf)
}

func (cc *COptClient) handleNext(i int, next []*pb.Blueprint, createconf bool) {
	if len(next) == 0 {
		return
	}

	for _, nxt := range next {
		if nxt != nil {
			i = cc.findorinsert(i, nxt, createconf)
		}
	}
}

func (cc *COptClient) findorinsert(i int, blp *pb.Blueprint, createconf bool) int {
	old := true
	for ; i < len(cc.Blueps); i++ {
		switch cc.Blueps[i].LearnedCompare(blp) {
		case 0:
			return i
		case 1:
			old = false
			continue
		case -1:
			if old { //This is an outdated blueprint.
				return i
			}
			cc.insert(i, blp, createconf)
			return i
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	cc.insert(i, blp, createconf)
	return i
}

func (cc *COptClient) insert(i int, blp *pb.Blueprint, createconf bool) {
	if createconf {
		cnf := cc.create(blp)
		cc.Confs = append(cc.Confs, cnf)

		for j := len(cc.Blueps) - 1; j > i; j-- {
			cc.Confs[j] = cc.Confs[j-1]
		}

		if len(cc.Blueps) != i+1 {
			cc.Confs[i] = cnf
		}
	}

	glog.V(3).Infof("Inserting next configuration with length %d at place %d\n", blp.Len(), i)

	cc.Blueps = append(cc.Blueps, blp)

	for j := len(cc.Blueps) - 1; j > i; j-- {
		cc.Blueps[j] = cc.Blueps[j-1]
	}

	if len(cc.Blueps) != i+1 {
		cc.Blueps[i] = blp
	}
}

func (cc *COptClient) create(blp *pb.Blueprint) *pb.Configuration {
	cnf, err := cc.mgr.NewConfiguration(blp.Add, majQuorum(blp), ConfTimeout)
	if err != nil {
		panic("could not get new config")
	}
	return cnf
}

func (cc *COptClient) createX(rids, cids []uint32, write bool) *pb.Configuration {
	x := pb.Difference(cids, rids)
	var q = len(cids)/2 + 1
	if write {
		if len(cids)-len(x) >= q {
			//We already have replies from a quorum.
			return nil
		}
		q = q - len(cids) + len(x)
	} else {
		//Read
		if len(cids)-len(x) >= len(cids)-q+1 {
			return nil
		}
	}
	cnf, err := cc.mgr.NewConfiguration(x, q, ConfTimeout)
	if err != nil {
		panic("could not get new config")
	}
	return cnf
}
