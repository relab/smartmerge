package smclient

import (
	"fmt"
	"time"

	pb "github.com/relab/smartMerge/proto"
	//"github.com/relab/smartMerge/rpc"
)

func (smc *SmClient) get() (rs *pb.State, cnt int) {
	cnt = 0
	cur := 0
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		read, err := smc.Confs[i].AReadS(&pb.AdvRead{uint32(smc.Blueps[i].Len())})
		cnt++
		cur = smc.handleNewCur(cur, read.Reply.GetCur())
		if err != nil && cur <= i {
			fmt.Println("error from AReadS: ", err)
			//No Quorum Available. Retry
			panic("Aread returned error")
			//return
		}

		smc.handleNext(i, read.Reply.GetNext())

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

		write, err := smc.Confs[i].AWriteS(&pb.AdvWriteS{rs,uint32(smc.Blueps[i].Len())})
		cnt++
		cur = smc.handleNewCur(cur, write.Reply.GetCur())
		if err != nil && cur <= i {
			fmt.Println("AWriteS returned error, ", err)
			panic("Error from ARead")
		}

		smc.handleNext(i, write.Reply.GetNext())
	}
	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return cnt
}

func (smc *SmClient) handleNewCur(cur int, newCur *pb.Blueprint) int {
	if newCur == nil {
		return cur
	}
	return smc.findorinsert(cur, newCur)
}

func (smc *SmClient) handleNext(i int, next []*pb.Blueprint) {
	for _, nxt := range next {
		if nxt != nil {
			i = smc.findorinsert(i, nxt)
		}
	}
}

func (smc *SmClient) findorinsert(i int, blp *pb.Blueprint) int {
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
			smc.insert(i, blp)
			return i
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	smc.insert(i, blp)
	return i
}

func (smc *SmClient) insert(i int, blp *pb.Blueprint) {
	cnf, err := smc.mgr.NewConfiguration(blp.Add, majQuorum(blp),2 * time.Second)
	if err != nil {
		panic("could not get new config")
	}

	smc.Blueps = append(smc.Blueps, blp)
	smc.Confs = append(smc.Confs, cnf)

	for j:= len(smc.Blueps)-1; j>i; j-- {
		smc.Blueps[j] = smc.Blueps[j-1]
		smc.Confs[j] = smc.Confs[j-1]
	} 

	if len(smc.Blueps) != i + 1 {
		smc.Blueps[i] = blp
		smc.Confs[i] = cnf
	}
}
