package smclient

import (
	"github.com/relab/smartMerge/rpc"
	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) get() (rs *pb.State) {
	cur := 0
	for i:=0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		st, next, newCur, err := smc.Confs[i].AReadS(smc.Blueps[cur])
		cur = smc.handleNewCur(cur, newCur)
		if err != nil {
			//No Quorum Available. Retry
			i--
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

func (smc *SmClient) set(rs *pb.State) {
	cur := 0
	for i:=0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		next, newCur, err := smc.Confs[i].AWriteS(rs, smc.Blueps[cur])
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
			if (blp).Compare(smc.Blueps[i]) == 1 {
				//Are equal
				return i
			}
			old = false
			continue
		case -1:
			if old { return i} //This is an outdated blueprint.
			smc.insert(i,blp)
			i++
			return i
		case 0:
			panic("blueprint not comparable")
		}
	}
	smc.insert(i,blp)
	i++
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

	blps := make([]*lat.Blueprint,len(smc.Blueps)+1)
	cnfs := make([]*rpc.Configuration,len(smc.Confs)+1)

	copy(blps, smc.Blueps[:i])
	copy(cnfs, smc.Confs[:i])

	blps[i] = blp
	cnfs[i] = cnf

	for ; i<len(smc.Blueps); i++ {
		blps[i+1] = smc.Blueps[i]
		cnfs[i+1] = smc.Confs[i]
	}

	smc.Blueps = blps
	smc.Confs = cnfs
}
