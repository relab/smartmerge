package consclient

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (cc *CClient) get() (rs *pb.State, cnt int) {
	cnt = 0
	cur := 0
	for i := 0; i < len(cc.Confs); i++ {
		if i < cur {
			continue
		}

		read, err := cc.Confs[i].CReadS(&pb.AdvRead{uint32(cc.Blueps[i].Len())})
		cnt++
		cur = cc.handleNewCur(cur, read.Reply.GetCur())
		if err != nil && cur <= i {
			glog.Errorln("error from CReadS: ", err)
			return nil, 0
			//return
		}

		if glog.V(6) {
			glog.Infoln("CReadS returned with replies from ", read.MachineIDs)
		}

		for _, next := range read.Reply.GetNext() {
			cc.handleNext(i, next)
		}
		
		if rs.Compare(read.Reply.GetState()) == 1 {
			rs = read.Reply.GetState()
		}
	}
	if cur > 0 {
		cc.Blueps = cc.Blueps[cur:]
		cc.Confs = cc.Confs[cur:]
	}
	return
}

func (cc *CClient) set(rs *pb.State) int {
	cnt := 0
	cur := 0
	for i := 0; i < len(cc.Confs); i++ {
		if i < cur {
			continue
		}

		write, err := cc.Confs[i].CWriteS(&pb.AdvWriteS{rs,uint32(cc.Blueps[i].Len())})
		cnt++
		cur = cc.handleNewCur(cur, write.Reply.GetCur())
		if err != nil {
			glog.Errorln("CWriteS returned error, ", err)
			return 0
		}
		if glog.V(6) {
			glog.Infoln("CWriteS returned, with replies from ", write.MachineIDs)
		}
		

		// This should never be more than one iteration. How to fix that?
		for _, next := range write.Reply.GetNext() {
			cc.handleNext(i, next)
		}
		
	}
	if cur > 0 {
		cc.Blueps = cc.Blueps[cur:]
		cc.Confs = cc.Confs[cur:]
	}
	return cnt
}

func (cc *CClient) handleNewCur(cur int, newCur *pb.Blueprint) int {
	if newCur == nil {
		return cur
	}
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Len(), cc.Blueps[cur].Len())
	}
	return cc.findorinsert(cur, newCur)
}

func (cc *CClient) handleNext(i int, next *pb.Blueprint) {
	if next != nil {
		glog.V(3).Infof("Found a next configurations.")
		i = cc.findorinsert(i, next)
	}
}

func (cc *CClient) findorinsert(i int, blp *pb.Blueprint) int {
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
			cc.insert(i, blp)
			return i
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	cc.insert(i, blp)
	return i
}

func (cc *CClient) insert(i int, blp *pb.Blueprint) {
	cnf, err := cc.mgr.NewConfiguration(blp.Add, majQuorum(blp),ConfTimeout)
	if err != nil {
		panic("could not get new config")
	}
	
	glog.V(3).Infoln("Inserting new configuration at place ", i)

	cc.Blueps = append(cc.Blueps, blp)
	cc.Confs = append(cc.Confs, cnf)

	for j:= len(cc.Blueps)-1; j>i; j-- {
		cc.Blueps[j] = cc.Blueps[j-1]
		cc.Confs[j] = cc.Confs[j-1]
	} 

	if len(cc.Blueps) != i + 1 {
		cc.Blueps[i] = blp
		cc.Confs[i] = cnf
	}
}
