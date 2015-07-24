package dynaclient

import (
	//"errors"
	"fmt"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"github.com/relab/smartMerge/rpc"
)

func (dc *DynaClient) Traverse(prop *lat.Blueprint, val []byte) ([]byte, int, error) {
	cnt := 0
	cur := 0
	rst := new(pb.State)
	for i := 0; i < len(dc.Confs); i++ {
		if i < cur {
			continue
		}

		if !prop.Equals(dc.Blueps[i]) {
			//Update Snapshot
			next, newCur, err := dc.Confs[i].GetOneN(dc.Blueps[i], prop)
			cnt++
			cur = dc.handleNewCur(i, newCur)
			if i < cur {
				continue
			}

			if err != nil {
				fmt.Println("Error from GetOneN")
				return nil, 0, err
			}

			//A possible optimization would combine this WriteN with the ReadS below
			newCur, err = dc.Confs[i].DWriteNSet([]*lat.Blueprint{next}, dc.Blueps[i])
			cnt++
			cur = dc.handleNewCur(i, newCur)
			if i < cur {
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteNSet")
				return nil, 0, err
			}

		}

		//ReadInView
		st, next, newCur, err := dc.Confs[i].DReadS(dc.Blueps[i])
		cnt++
		cur = dc.handleNewCur(i, newCur)
		if i < cur {
			continue
		}
		if err != nil {
			fmt.Println("Error from DReadS")
			return nil, 0, err
		}

		prop = dc.handleNext(i, next, prop)
		if rst.Compare(st) == 1 {
			rst = st
		}

		if len(next) == 0 {
			//WriteInView
			wst := dc.WriteValue(val, rst)
			next, newCur, err := dc.Confs[i].DWriteS(wst, dc.Blueps[i])
			cnt++
			cur = dc.handleNewCur(i, newCur)
			if i < cur {
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
			cnt++
			cur = dc.handleNewCur(i, newCur)
			if err != nil {
				fmt.Println("Error from DWriteNSet")
				return nil, 0, err
			}
			continue
		}
	}

	if i := len(dc.Confs) - 1; i > cur {
		dc.Confs[i].DSetCur(dc.Blueps[i])
		cnt++
		cur = i
	}

	dc.Blueps = dc.Blueps[cur:]
	dc.Confs = dc.Confs[cur:]

	if val == nil {
		return nil, cnt, nil
	}
	return rst.Value, cnt, nil
}

func (dc *DynaClient) handleNewCur(cur int, newCur *lat.Blueprint) int {
	if newCur == nil {
		return cur
	}
	cur, remove := dc.findorinsert(cur, newCur)
	if remove {
		for ; cur < len(dc.Blueps)-1; cur++ {
			if dc.Blueps[cur+1].Compare(dc.Blueps[cur]) == 0 {
				dc.Blueps[cur+1] = dc.Blueps[cur]
			} else {
				break
			}
		}
	}
	return cur

}

func (dc *DynaClient) handleNext(i int, next []*lat.Blueprint, prop *lat.Blueprint) *lat.Blueprint {
	for _, nxt := range next {
		if nxt != nil {
			i, _ = dc.findorinsert(i, nxt)
			prop = prop.Merge(nxt)
		}
	}
	return prop
}

func (dc *DynaClient) findorinsert(i int, blp *lat.Blueprint) (index int, old bool) {
	old = true
	for ; i < len(dc.Blueps); i++ {
		switch (dc.Blueps[i]).Compare(blp) {
		case 1, 0:
			if blp.Compare(dc.Blueps[i]) == 1 {
				//Are equal
				//fmt.Println("Blueprints equal, return")
				return i, true
			}
			old = false
			continue
		case -1:
			if old { //This is an outdated blueprint.
				return i, false
			}
			dc.insert(i, blp)
			return i, false
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	dc.insert(i, blp)
	return i, false
}

func (dc *DynaClient) insert(i int, blp *lat.Blueprint) {
	cnf, err := dc.mgr.NewConfiguration(blp.Ids(), majQuorum(blp))
	if err != nil {
		panic("could not get new config")
	}

	if i >= len(dc.Blueps) {
		dc.Blueps = append(dc.Blueps, blp)
		dc.Confs = append(dc.Confs, cnf)
		return
	}

	blps := make([]*lat.Blueprint, len(dc.Blueps)+1)
	cnfs := make([]*rpc.Configuration, len(dc.Confs)+1)

	copy(blps, dc.Blueps[:i])
	copy(cnfs, dc.Confs[:i])

	blps[i] = blp
	cnfs[i] = cnf

	for ; i < len(dc.Blueps); i++ {
		blps[i+1] = dc.Blueps[i]
		cnfs[i+1] = dc.Confs[i]
	}

	dc.Blueps = blps
	dc.Confs = cnfs
}

func (dc *DynaClient) WriteValue(val []byte, st *pb.State) *pb.State {
	if val == nil {
		return st
	}
	return &pb.State{Value: val, Timestamp: st.Timestamp + 1, Writer: dc.ID}
}
