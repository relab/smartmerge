package dynaclient

import (
	//"errors"
	"fmt"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/rpc"
)

func (dc *DynaClient) OrgTraverse(prop *lat.Blueprint, val []byte) ([]byte, int, error) {
	cnt := 0
	rst := new(pb.State)
	for i := 0; i < len(dc.Confs); i++ {
		if prop != nil && !prop.Equals(dc.Blueps[i]) {
			//Update Snapshot
/*			if dc.Blueps[i].Compare(prop) != 1 {
				fmt.Println("target blueprint is not greater then current blueprint")
				return nil, cnt, errors.New("target not comparable to current")
			}
*/			
			next, newCur, err := dc.Confs[i].GetOneN(dc.Blueps[i], prop)
			//fmt.Println("invoke getone")
			cnt++
			restart := dc.abortonNewCur(newCur)
			if restart {
				prop = prop.Merge(newCur)
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from GetOneN")
				return nil, 0, err
			}

			newCur, err = dc.Confs[i].DWriteNSet([]*lat.Blueprint{next}, dc.Blueps[i])
			//fmt.Println("invoke writeN")
			cnt++
			restart = dc.abortonNewCur(newCur)
			if restart {
				prop = prop.Merge(newCur)
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteNSet")
				return nil, 0, err
			}
		}

		//ReadInView:
		st, _, newCur, err := dc.Confs[i].DReadS(dc.Blueps[i], nil)
		//fmt.Println("invoke readS")
		cnt++
		restart := dc.abortonNewCur(newCur)
		if restart {
			prop = prop.Merge(newCur)
			i = -1
			continue
		}
		if err != nil {
			fmt.Println("Error from DReadS")
			return nil, 0, err
		}

		//Using DWriteS here is a shortcut, but I think it works just fine.
		next, newCur, err := dc.Confs[i].DWriteS(nil, dc.Blueps[i])
		//fmt.Println("invoke readN")
		cnt++
		restart = dc.abortonNewCur(newCur)
		if restart {
			prop = prop.Merge(newCur)
			i = -1
			continue
		}
		if err != nil {
			fmt.Println("Error from DReadN")
			return nil, 0, err
		}

		prop = dc.handleNext(i, next, prop)
		if rst.Compare(st) == 1 {
			rst = st
		}

		if len(next) == 0 {
			//WriteInView
			wst := dc.WriteValue(val, rst)
			_, newCur, err = dc.Confs[i].DWriteS(wst, dc.Blueps[i])
			//fmt.Println("invoke writeS")
			cnt++
			restart = dc.abortonNewCur(newCur)
			if restart {
				prop = prop.Merge(newCur)
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteS")
				return nil, 0, err
			}

			//Using DWriteS here is a shortcut, but I think it works just fine.
			next, newCur, err := dc.Confs[i].DWriteS(nil, dc.Blueps[i])
			//fmt.Println("invoke readN")
			cnt++
			restart = dc.abortonNewCur(newCur)
			if restart {
				prop = prop.Merge(newCur)
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteS")
				return nil, 0, err
			}

			prop = dc.handleNext(i, next, prop)

		}

		if len(next) > 0 {
			newCur, err = dc.Confs[i].DWriteNSet(next, dc.Blueps[i])
			//fmt.Println("invoke writeN")
			cnt++
			restart = dc.abortonNewCur(newCur)
			if restart {
				prop = prop.Merge(newCur)
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteNSet")
				return nil, 0, err
			}
			continue
		}
	}

	i := len(dc.Confs) -1
	if i > 0  {
		dc.Confs[i].DSetCur(dc.Blueps[i])
		//fmt.Println("setcur")
		cnt++
	}

	dc.Blueps = dc.Blueps[i:]
	dc.Confs = dc.Confs[i:]

	if val == nil {
		return rst.Value, cnt, nil
	}
	return nil, cnt, nil
}

func (dc *DynaClient) abortonNewCur(newCur *lat.Blueprint) bool {
	if newCur.Compare(dc.Blueps[0]) == 1 {
		return false
	}
	found := false
	cur := 1
	for ; cur < len(dc.Blueps); cur++ {
		if newCur.Compare(dc.Blueps[cur]) == 1 {
			if dc.Blueps[cur].Compare(newCur) == 1 {
				found = true
			}
			break
		}
	}

	if found {
		dc.Blueps = dc.Blueps[cur:cur+1]
		dc.Confs = dc.Confs[cur:cur+1]
	} else {
		cnf, err := dc.mgr.NewConfiguration(newCur.Ids(), majQuorum(newCur))
		if err != nil {
			panic("could not get new config")
		}
		dc.Blueps = []*lat.Blueprint{newCur}
		dc.Confs = []*rpc.Configuration{cnf}
	}
	return true
}