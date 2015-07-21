package smclient

import (
	"fmt"
	"errors"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) Reconf(prop *lat.Blueprint) error {
	prop = smc.lagree(prop)
	//fmt.Println("LA returned ", prop)
	
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(smc.Blueps[0]) == 1 {
		return nil
	}

	if prop.Compare(smc.Blueps[0]) == 0 {
		panic("Lattice agreement returned an uncomparable blueprint")
	}
	
	if prop.Compare(smc.Blueps[len(smc.Blueps)-1]) == 1 {
		prop = smc.Blueps[len(smc.Blueps)-1]
	}

	if len(prop.Ids()) == 0 {
		return errors.New("Abort before proposing unacceptable configuration.")
	}

	cur := 0
	las := new(lat.Blueprint)
	rst := new(pb.State)
	for i := 0; i < len(smc.Confs); i++ {
		//fmt.Println("Start State Transfer")
		if i < cur {
			continue
		}

		st, newlas, next, newCur, err := smc.Confs[i].AWriteN(prop, smc.Blueps[i])
		//fmt.Println("NewCur was: ", newCur)
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
		//fmt.Println("Len Blueps, Confs: ", len(smc.Blueps), len(smc.Confs))
		//fmt.Println("Cur has index ", cur)
	}

	if i := len(smc.Confs)-1; i > cur {
		smc.Confs[i].AWriteS(rst, smc.Blueps[i])
		smc.Confs[i].LAProp(smc.Blueps[i], las)
		smc.Confs[i].SetCur(smc.Blueps[i])
		cur = i
	}

	smc.Blueps = smc.Blueps[cur:]
	smc.Confs = smc.Confs[cur:]

	return nil
}

func (smc *SmClient) lagree(prop *lat.Blueprint) *lat.Blueprint {
	//fmt.Println("Start LA")
	cur := 0
	prop = prop.Merge(smc.Blueps[0])
	for i := 0; i < len(smc.Confs); i++ {
		//fmt.Println("LA in conf ", smc.Blueps[0], " with index ", i)
		if i < cur {
			continue
		}

		la, next, newCur, err := smc.Confs[i].LAProp(smc.Blueps[i], prop)
		cur = smc.handleNewCur(cur, newCur)
		if err != nil {
			fmt.Println("LA prop returned error: ", err)
			i--
		}

		if la != nil && !prop.Equals(la) {
			//fmt.Println("LA prop returned new LA state ", la)
			prop = la
			i--
		}

		smc.handleNext(i, next)
	}
	if cur > 0 {
		smc.Blueps = smc.Blueps[cur:]
		smc.Confs = smc.Confs[cur:]
	}
	return prop
}
