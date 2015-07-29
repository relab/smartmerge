package smclient

import (
	"fmt"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/rpc"
)

func (smc *SmClient) get() (rs *pb.State, cnt int) {
	cnt = 0
	cur := 0
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		st, next, newCur, err := smc.Confs[i].AReadS(smc.Blueps[i], smc.Confs[cur].ID())
		cnt++
		cur = smc.handleNewCur(cur, newCur)
		if err != nil {
			fmt.Println("error from AReadS: ", err)
			//No Quorum Available. Retry
			i--
			//return
		}

		smc.handleNext(i, next)

		if rs.Compare(st) == 1 {
			rs = st
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

		next, newCur, err := smc.Confs[i].AWriteS(rs,smc.Confs[cur].ID(), smc.Blueps[i])
		cnt++
		cur = smc.handleNewCur(cur, newCur)
		if err != nil {
			i--
		}

		smc.handleNext(i, next)
	}
	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return cnt
}

func (smc *SmClient) handleNewCur(cur int, newCur *lat.Blueprint) int {
	if newCur == nil {
		return cur
	}
	return smc.findorinsert(cur, newCur)
}

func (smc *SmClient) handleNext(i int, next []*lat.Blueprint) {
	for _, nxt := range next {
		if nxt != nil {
			i = smc.findorinsert(i, nxt)
		}
	}
}

func (smc *SmClient) findorinsert(i int, blp *lat.Blueprint) int {
	old := true
	for ; i < len(smc.Blueps); i++ {
		switch (smc.Blueps[i]).Compare(blp) {
		case 1:
			if blp.Compare(smc.Blueps[i]) == 1 {
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
			smc.insert(i, blp)
			return i
		case 0:
			panic("blueprint not comparable")
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	smc.insert(i, blp)
	return i
}

func (smc *SmClient) insert(i int, blp *lat.Blueprint) {
	cnf, err := smc.mgr.NewConfiguration(blp.Ids(), majQuorum(blp))
	if err != nil {
		panic("could not get new config")
	}

	if i >= len(smc.Blueps) {
		smc.Blueps = append(smc.Blueps, blp)
		smc.Confs = append(smc.Confs, cnf)
		return
	}

	blps := make([]*lat.Blueprint, len(smc.Blueps)+1)
	cnfs := make([]*rpc.Configuration, len(smc.Confs)+1)

	copy(blps, smc.Blueps[:i])
	copy(cnfs, smc.Confs[:i])

	blps[i] = blp
	cnfs[i] = cnf

	for ; i < len(smc.Blueps); i++ {
		blps[i+1] = smc.Blueps[i]
		cnfs[i+1] = smc.Confs[i]
	}

	smc.Blueps = blps
	smc.Confs = cnfs
}
