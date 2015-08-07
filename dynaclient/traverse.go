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
		var curprop *lat.Blueprint
		if prop != nil && !prop.Equals(dc.Blueps[i]) {
			//Update Snapshot
			next, newCur, err := dc.Confs[i].GetOneN(dc.Blueps[i], prop)
			//fmt.Println("invoke getone")
			cnt++
			cur = dc.handleNewCur(cur, i, newCur)
			prop = prop.Merge(newCur)
			if i < cur {
				continue
			}

			if err != nil {
				fmt.Println("Error from GetOneN")
				return nil, 0, err
			}
			
			curprop = next

		}

		//ReadInView:
		st, next, newCur, err := dc.Confs[i].DReadS(dc.Blueps[i], curprop)
		//fmt.Println("invoke readS")
		cnt++
		cur = dc.handleNewCur(cur, i, newCur)
		if prop != nil || len(next) > 0 {
			prop = prop.Merge(newCur)
		}
		if i < cur {
			continue
		}
		if err != nil {
			fmt.Println("Error from DReadS")
			return nil, 0, err
		}

		/*		for _,nxt := range next {
					if dc.Blueps[i].Compare(nxt) != 1{
						fmt.Println("Returned next, that is not greater than this")
						panic("Next not comparable.")
					}
				}
		*/
		prop = dc.handleNext(i, next, prop)
		if rst.Compare(st) == 1 {
			rst = st
		}

		/*
			if i > 40 {
				fmt.Println("Did too many loops, there is a problem/race condition.")
				fmt.Printf("Currently at position %d in slice.\n", i)
				fmt.Printf("Slice length is %d.\n", len(dc.Blueps))
				if dc.ID == uint32(1) || dc.ID == uint32(6) {
					fmt.Println("Blueprints slice is:")
					for _,bl := range dc.Blueps {
						fmt.Printf("   %v\n", bl.Rem)
					}
					panic("Too many blueprints")
				}

				return nil, cnt, errors.New("Race condition")
			}
		*/
		if len(next) == 0 {
			//WriteInView
			wst := dc.WriteValue(val, rst)
			next, newCur, err = dc.Confs[i].DWriteS(wst, dc.Blueps[i])
			//fmt.Println("invoke writeS")
			cnt++
			cur = dc.handleNewCur(cur, i, newCur)
			if prop != nil || len(next) > 0 {
				prop = prop.Merge(newCur)
			}
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
			//fmt.Println("invoke writeN")
			cnt++
			cur = dc.handleNewCur(cur, i, newCur)
			if prop != nil {
				prop = prop.Merge(newCur)
			}
			if err != nil {
				fmt.Println("Error from DWriteNSet")
				return nil, 0, err
			}
			continue
		}
	}

	if i := len(dc.Confs) - 1; i > cur {
		dc.Confs[i].DSetCur(dc.Blueps[i])
		//fmt.Println("setcur")
		cnt++
		cur = i
	}

	dc.Blueps = dc.Blueps[cur:]
	dc.Confs = dc.Confs[cur:]

	if val == nil {
		return rst.Value, cnt, nil
	}
	return nil, cnt, nil
}

func (dc *DynaClient) handleNewCur(cur int, i int, newCur *lat.Blueprint) int {
	if newCur == nil {
		return cur
	}
	if newCur.Compare(dc.Blueps[i]) == 1 {
		return cur
	}
	cur, remove := dc.findorinsert(i, newCur)
	if remove {
		for ; cur < len(dc.Blueps)-1; cur++ {
			if dc.Blueps[cur+1].Compare(dc.Blueps[cur]) == 0 {
				dc.Blueps[cur+1] = dc.Blueps[cur]
				dc.Confs[cur+1] = dc.Confs[cur]
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
