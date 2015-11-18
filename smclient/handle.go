package smclient

import (
	"github.com/golang/glog"

	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) SetNewCur(cur int) {
	if cur >= len(smc.Blueps) {
		glog.Fatalln("Index for new cur out of bound.")
	}

	if cur == 0 {
		return
	}

	smc.Blueps = smc.Blueps[cur:]
}

func (smc *SmClient) HandleOneCur(cur int, newCur *pb.Blueprint) int {
	if newCur == nil {
		return cur
	}
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Len(), smc.Blueps[cur].Len())
	}
	return smc.findorinsert(cur, newCur)
}

func (smc *SmClient) HandleNewCur(cur int, newCur *pb.ConfReply) int {
	if newCur == nil {
		return cur
	}
	smc.HandleNext(cur, newCur.Next)
	if newCur.Cur == nil {
		return cur
	}
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Cur.Len(), smc.Blueps[cur].Len())
	}

	return smc.findorinsert(cur, newCur.Cur)
}

func (smc *SmClient) HandleNext(i int, next []*pb.Blueprint) {
	if len(next) == 0 {
		return
	}

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
	smc.insert(i, blp)
	return i
}

func (smc *SmClient) insert(i int, blp *pb.Blueprint) {
	glog.V(3).Infof("Inserting new blueprint with length %d at place %d\n", blp.Len(), i)

	smc.Blueps = append(smc.Blueps, blp)

	for j := len(smc.Blueps) - 1; j > i; j-- {
		smc.Blueps[j] = smc.Blueps[j-1]
	}

	if len(smc.Blueps) != i+1 {
		smc.Blueps[i] = blp
	}
}
