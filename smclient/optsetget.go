package smclient

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmOptClient) get() (rs *pb.State, cnt int) {
	cnt = 0
	cur := 0
	rid := make([]uint32,0)
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}
		cnt++
		var cnf *pb.Configuration
		
		
		if i > 0 {
			cnf = smc.createX(rid, smc.Blueps[i].Ids(), false)
			if cnf == nil {
				glog.V(4).Infoln("We can skip contacting next configuration.")
				continue
			}
			smc.Confs[i] = cnf
		} else { cnf = smc.Confs[0] }

		read, err := cnf.AReadS(&pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[cur].Len())})
		if err != nil && (read == nil || read.Reply == nil)  {
			glog.Errorln("error from AReadS: ", err)
			//No Quorum Available. Retry
			return nil, 0
		}
		if glog.V(6) {
			glog.Infoln("AReadS returned with replies from ", read.MachineIDs)
		}
		
		cur = smc.handleNewCur(cur, read.Reply.GetCur(), false)

		smc.handleNext(i, read.Reply.GetNext(), false)

		if rs.Compare(read.Reply.GetState()) == 1 {
			rs = read.Reply.GetState()
		}
		
		if len(smc.Blueps) > i+1 {
			rid = pb.Union(rid, read.MachineIDs)
		}
		
	}
	if cur > 0 {
		smc.Confs[0] = smc.create(smc.Blueps[cur])
		smc.Blueps = smc.Blueps[cur:]
	}
	return
}

func (smc *SmOptClient) set(rs *pb.State) int {
	cnt := 0
	cur := 0
	rid := make([]uint32,0)
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		var cnf *pb.Configuration

		cnt++
		if i > 0 {
			cnf = smc.createX(rid, smc.Blueps[i].Ids(), true)
			if cnf == nil {
				glog.V(4).Infoln("We can skip contacting next configuration.")
				continue
			}
		} else { cnf = smc.Confs[0] }

		write, err := cnf.AWriteS(&pb.WriteS{rs, &pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[cur].Len())}})
		if err != nil {
			glog.Errorln("AWriteS returned error, ", err)
			return 0
		}
		if glog.V(6) {
			glog.Infoln("AWriteS returned, with replies from ", write.MachineIDs)
		}

		cur = smc.handleNewCur(cur, write.Reply.GetCur(), false)
		smc.handleNext(i, write.Reply.GetNext(), false)
		
		if len(smc.Blueps) > i+1 {
			rid = pb.Union(rid, write.MachineIDs)
		}
	}

	if cur > 0 {
		smc.Confs[0] = smc.create(smc.Blueps[cur])
		smc.Blueps = smc.Blueps[cur:]
	}

	return cnt
}

func (smc *SmOptClient) handleNewCur(cur int, newCur *pb.ConfReply, createconf bool) int {
	if newCur == nil {
		return cur
	}
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Cur.Len(), smc.Blueps[cur].Len())
	}
	return smc.findorinsert(cur, newCur.Cur, createconf)
}

func (smc *SmOptClient) handleNext(i int, next []*pb.Blueprint, createconf bool) {
	if len(next) == 0 {
		return
	}
	
	for _, nxt := range next {
		if nxt != nil {
			i = smc.findorinsert(i, nxt, createconf)
		}
	}
}

func (smc *SmOptClient) findorinsert(i int, blp *pb.Blueprint, createconf bool) int {
	old := true
	for ; i < len(smc.Blueps); i++ {
		switch smc.Blueps[i].LearnedCompare(blp) {
		case 0:
			return i
		case 1:
			old = false
			continue
		case -1:
			if old { //This is an outdated blueprint.
				return i
			}
			smc.insert(i, blp, createconf)
			return i
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	smc.insert(i, blp, createconf)
	return i
}

func (smc *SmOptClient) insert(i int, blp *pb.Blueprint, createconf bool) {
	if createconf {	
		cnf := smc.create(blp)
		smc.Confs = append(smc.Confs, cnf)

		for j := len(smc.Blueps) - 1; j > i; j-- {
			smc.Confs[j] = smc.Confs[j-1]
		}

		if len(smc.Blueps) != i+1 {
			smc.Confs[i] = cnf
		}
	}

	glog.V(3).Infof("Inserting next configuration with length %d at place %d\n", blp.Len(), i)

	smc.Blueps = append(smc.Blueps, blp)

	for j := len(smc.Blueps) - 1; j > i; j-- {
		smc.Blueps[j] = smc.Blueps[j-1]
	}

	if len(smc.Blueps) != i+1 {
		smc.Blueps[i] = blp
	}
}

func (smc *SmOptClient) create(blp *pb.Blueprint) (*pb.Configuration) {
	cnf, err := smc.mgr.NewConfiguration(blp.Add, majQuorum(blp), ConfTimeout)
	if err != nil {
		panic("could not get new config")
	}
	return cnf
}

func (smc *SmOptClient) createX(rids, cids []uint32, write bool) *pb.Configuration {
	x := pb.Difference(cids, rids)
	var q = len(cids)/2 +1
	if write {
		if len(cids) - len(x) >= q {
			//We already have replies from a quorum.
			return nil
		}
		q = q - len(cids) + len(x)
	} else {
		//Read
		if len(cids) - len(x) >= len(cids) - q + 1 {
			return nil
		}
	}
	cnf, err := smc.mgr.NewConfiguration(x, q, ConfTimeout)
	if err != nil {
		panic("could not get new config")
	}
	return cnf
}