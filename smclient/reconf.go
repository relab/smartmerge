package smclient

import (
	"errors"

	"github.com/relab/smartMerge/rpc"
	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) reconf(prop *lat.Blueprint) error {
	prop = smc.lagree(prop)
	if len(nextCur.Ids())= 0 {
		return errors.New("Abort because of unacceptable configuration.")
	}
	cur := 0
	las := new(lat.Blueprint)
	rst := new(pb.State)
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		st, newlas, next, newCur, err := smc.Confs[i].AWriteN(prop, smc.Blueps[cur])
		cur = smc.handleNewCur(cur, newCur)
		if err != nil {
			//Should logg this for debugging
			i--
		}

		smc.handleNext(i, next)

	}


}

func (smc *SmClient) lagree(prop *lat.Blueprint) *lat.Blueprint {
	cur := 0
	tmp := prop.Merge(*smc.Blueps[0])
	prop = &tmp
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		la, next, newCur, err := smc.Confs[i].LAProp(smc.Blueps[cur], prop)
		cur = smc.handleNewCur(cur, newCur)
		if err != nil {
			i--
		}

		if la != nil {
			prop = la
			i--
		}

		smc.handleNext(i+1, next)
	}
	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
}

