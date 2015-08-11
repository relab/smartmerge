package consclient

import (
	"fmt"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/rpc"
)

func (cc *CClient) get() (rs *pb.State, cnt int) {
	cnt = 0
	cur := 0
	for i := 0; i < len(cc.Confs); i++ {
		if i < cur {
			continue
		}

		st, next, newCur, err := cc.Confs[i].CReadS(cc.Blueps[i], cc.Confs[cur].ID(), nil)
		cnt++
		cur = cc.handleNewCur(cur, newCur)
		if err != nil && cur <= i {
			fmt.Println("error from AReadS: ", err)
			//No Quorum Available. Retry
			panic("Aread returned error")
			//return
		}

		cc.handleNext(i, next)

		if rs.Compare(st) == 1 {
			rs = st
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

		next, newCur, err := cc.Confs[i].CWriteS(rs,cc.Blueps[i], cc.Confs[cur].ID())
		cnt++
		cur = cc.handleNewCur(cur, newCur)
		if err != nil && cur <= i {
			fmt.Println("CWriteS returned error, ", err)
			panic("Error from CRead")
		}

		cc.handleNext(i, next)
	}
	if cur > 0 {
		cc.Blueps = cc.Blueps[cur:]
		cc.Confs = cc.Confs[cur:]
	}
	return cnt
}

func (cc *CClient) handleNewCur(cur int, newCur *lat.Blueprint) int {
	if newCur == nil {
		return cur
	}
	return cc.findorinsert(cur, newCur)
}

func (cc *CClient) handleNext(i int, next *lat.Blueprint) {
	if next != nil {
		i = cc.findorinsert(i, next)
	}
}

func (cc *CClient) findorinsert(i int, blp *lat.Blueprint) int {
	old := true
	for ; i < len(cc.Blueps); i++ {
		switch cc.Blueps[i].Compare(blp) {
		case 1:
			if blp.Compare(cc.Blueps[i]) == 1 {
				//Are equal
				//fmt.Println("Blueprints equal, return")
				return i
			}
			old = false
			continue
		case -1:
			if old { //This is an outdated blueprint.
				return i
			}
			cc.insert(i, blp)
			return i
		case 0:
			panic("blueprint not comparable")
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	cc.insert(i, blp)
	return i
}

func (cc *CClient) insert(i int, blp *lat.Blueprint) {
	cnf, err := cc.mgr.NewConfiguration(blp.Ids(), majQuorum(blp))
	if err != nil {
		panic("could not get new config")
	}

	if i >= len(cc.Blueps) {
		cc.Blueps = append(cc.Blueps, blp)
		cc.Confs = append(cc.Confs, cnf)
		return
	}

	blps := make([]*lat.Blueprint, len(cc.Blueps)+1)
	cnfs := make([]*rpc.Configuration, len(cc.Confs)+1)

	copy(blps, cc.Blueps[:i])
	copy(cnfs, cc.Confs[:i])

	blps[i] = blp
	cnfs[i] = cnf

	for ; i < len(cc.Blueps); i++ {
		blps[i+1] = cc.Blueps[i]
		cnfs[i+1] = cc.Confs[i]
	}

	cc.Blueps = blps
	cc.Confs = cnfs
}
