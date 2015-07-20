package smclient

import (
	"errors"

	"github.com/relab/smartMerge/rpc"
	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) reconf(prop *lat.Blueprint) error {
	prop = smc.lagree(prop)
	if prop.Compare(smc.Blueps[0]) == -1 {
		return nil
	}
	if prop.Equals(smc.Blueps[0]) {
		return nil
	}
	
	if prop.Compare(smc.Blueps[0]) == 0 {
		panic("Lattice agreement returned an uncomparable blueprint")
	}
	
	if prop.Compare(smc.Blueps[len(smc.Blueps)-1]) == -1 {
		prop = smc.Blueps[len(smc.Blueps)-1]
	}
	
	if len(prop.Ids())= 0 {
		return errors.New("Abort before proposing unacceptable configuration.")
	}
	
	cur := 0
	las := new(lat.Blueprint)
	rst := new(pb.State)
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		st, newlas, next, newCur, err := smc.Confs[i].AWriteN(prop, smc.Blueps[i])
		cur = smc.handleNewCur(cur, newCur)
		if err != nil {
			//Should logg this for debugging
			i--
		}

		smc.handleNext(i, next)
		las = las.Merge(newlas)
		if rst.Compare(st) == 1 {
			rst = st
		}
		
		prop = smc.Blueps[len(smc.Blueps)-1]
	}
	
	if i := len(smc.Confs); i > cur {
		smc.Confs[i].WriteS(rst, smc.Blueps[i])
		smc.Confs[i].LAProp(smc.Blueps[i], las)
		smc.Confs[i].NewCur(smc.Blueps[i])
		cur = i
	}
	
	smc.Blueps = smc.Blueps[cur:]
	smc.Confs = smc.Confs[cur:]
	
	return nil
}

func (smc *SmClient) lagree(prop *lat.Blueprint) *lat.Blueprint {
	cur := 0
	tmp := prop.Merge(*smc.Blueps[0])
	prop = &tmp
	for i := 0; i < len(smc.Confs); i++ {
		if i < cur {
			continue
		}

		la, next, newCur, err := smc.Confs[i].LAProp(smc.Blueps[i], prop)
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

