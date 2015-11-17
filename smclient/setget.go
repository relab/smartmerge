package smclient

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) get() (rs *pb.State, cnt int) {
	cnt = 0
	cur := 0
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		read, err := smc.Confs[i].AReadS(&pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[cur].Len())})
		cnt++
		if err != nil {
			glog.Errorln("error from AReadS: ", err)
			//No Quorum Available. Retry
			return nil, 0
		}
		if glog.V(6) {
			glog.Infoln("AReadS returned with replies from ", read.MachineIDs)
		}
		cur = smc.handleNewCur(cur, read.Reply.GetCur(), true)

		if rs.Compare(read.Reply.GetState()) == 1 {
			rs = read.Reply.GetState()
		}
	}
	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return
}

func (smc *SmClient) set(rs *pb.State) int {
	cnt := 0
	cur := 0
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		write, err := smc.Confs[i].AWriteS(&pb.WriteS{rs, &pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[cur].Len())}})
		cnt++
		if err != nil {
			glog.Errorln("AWriteS returned error, ", err)
			return 0
		}
		if glog.V(6) {
			glog.Infoln("AWriteS returned, with replies from ", write.MachineIDs)
		}

		cur = smc.handleNewCur(cur, write.Reply, true)
	}
	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return cnt
}

func (smc *SmClient) handleNewCur(cur int, newCur *pb.ConfReply, createconf bool) int {
	if newCur == nil {
		return cur
	}
	smc.handleNext(cur, newCur.Next, createconf)
	if newCur.Cur == nil {
		return cur
	}
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Cur.Len(), smc.Blueps[cur].Len())
	}

	return smc.findorinsert(cur, newCur.Cur, createconf)
}

func (smc *SmClient) handleNext(i int, next []*pb.Blueprint, createconf bool) {
	if len(next) == 0 {
		return
	}

	for _, nxt := range next {
		if nxt != nil {
			i = smc.findorinsert(i, nxt, createconf)
		}
	}
}

func (smc *SmClient) findorinsert(i int, blp *pb.Blueprint, createconf bool) int {
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
	smc.insert(i, blp, createconf)
	return i
}

func (smc *SmClient) insert(i int, blp *pb.Blueprint, createconf bool) {
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

func (smc *SmClient) create(blp *pb.Blueprint) *pb.Configuration {
	cnf, err := smc.mgr.NewConfiguration(blp.Ids(), blp.Quorum(), ConfTimeout)
	if err != nil {
		glog.Fatalln("could not get new config")
	}
	return cnf
}
