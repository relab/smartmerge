package dynaclient

import (
	//"errors"
	"fmt"
	"time"

	pb "github.com/relab/smartMerge/proto"
)

func (dc *DynaClient) Traverse(prop *pb.Blueprint, val []byte, regular bool) ([]byte, int, error) {
	cnt := 0
	rst := new(pb.State)
	for i := 0; i < len(dc.Confs); i++ {
		var curprop *pb.Blueprint
		if prop != nil && !prop.Equals(dc.Blueps[i]) {
			//Update Snapshot
			getOne, err :=  dc.Confs[i].GetOneN(&pb.GetOne{
				CurC: uint32(dc.Blueps[i].Len()), 
				Next: prop,
			})
			cnt++
			isnew := dc.handleNewCur(i, getOne.Reply.GetCur())
			if isnew {
				prop = prop.Merge(getOne.Reply.GetCur())
				i = -1
				continue
			}
			
			if err != nil {
				fmt.Println("Error from GetOneN")
				return nil, 0, err
			}
			
			curprop = getOne.Reply.GetNext()

		}

		//ReadInView:
		read, err := dc.Confs[i].DReadS(
			&pb.DRead{
				CurC: uint32(dc.Blueps[i].Len()), 
				Prop: curprop,
			}
		)
		cnt++
		isnew := dc.handleNewCur(i, read.Reply.GetCur())
		if isnew {
			if prop != nil {
				prop = prop.Merge(read.Reply.GetCur())
			}
			i = -1
			continue
		}

		if err != nil {
			fmt.Println("Error from DReadS")
			return nil, 0, err
		}

		next := read.Reply.GetNext()
		prop = dc.handleNext(i, next, prop)
		if rst.Compare(read.Reply.GetState()) == 1 {
			rst = read.Reply.GetState()
		}

		if len(next) == 0 && !regular {
			
			//WriteInView
			wst := dc.WriteValue(val, rst)
			write, err := dc.Confs[i].DWriteS(
				&pb.AdvWriteS{
					State: wst,
					CurC: uint32(dc.Blueps[i].Len()),
				}
			)
			
			cnt++
			isnew = dc.handleNewCur(i, write.Reply.GetCur())
			if isnew {
				if prop != nil {
					prop = prop.Merge(write.Reply.GetCur())
				}
				i = -1
				continue
			}
		
			if err != nil {
				fmt.Println("Error from DWriteS")
				return nil, 0, err
			}
			
			next = write.Reply.GetNext()
			prop = dc.handleNext(i, next, prop)
		}

		if len(next) > 0 {
			regular = false
			writeN, err := dc.Confs[i].DWriteNSet(&pb.DWriteN{uint32(dc.Blueps[i].Len()),next})
			//fmt.Println("invoke writeN")
			cnt++
			isnew = dc.handleNewCur(i, writeN.Reply.GetCur())
			if isnew {
				if prop != nil {
					prop = prop.Merge(writeN.Reply.GetCur())
				}
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

	//TODO: Optimization: Integrate this into the writeS above.
	if i:= len(dc.Confs)-1 ; i>0  {
		dc.Confs[i].DSetCur(&pb.NewCur{dc.Blueps[i], uint32(dc.Blueps[i].Len())})
		//fmt.Println("setcur")
		cnt++
		
		dc.Blueps = dc.Blueps[i:]
		dc.Confs = dc.Confs[i:]
	}

	

	if val == nil {
		return rst.Value, cnt, nil
	}
	return nil, cnt, nil
}

func (dc *DynaClient) handleNewCur(i int, newCur *pb.Blueprint) bool {
	if newCur == nil {
		return false
	}
	if newCur.Compare(dc.Blueps[i]) == 1 {
		return false
	}
	
	cnf, err := dc.mgr.NewConfiguration(newCur.Add, majQuorum(newCur), 2 *  time.Second)
	if err != nil {
		panic("could not get new config")
	}
	
	dc.Blueps = make([]*pb.Blueprint,1,5)
	dc.Confs = make([]*pb.Configuration,1,5)
	dc.Blueps[0] = newCur
	dc.Confs[0] = cnf
	
	return true
	
}

func (dc *DynaClient) handleNext(i int, next []*pb.Blueprint, prop *pb.Blueprint) *pb.Blueprint {
	for _, nxt := range next {
		if nxt != nil {
			dc.findorinsert(i, nxt)
			prop = prop.Merge(nxt)
		}
	}
	return prop
}

func (dc *DynaClient) findorinsert(i int, blp *pb.Blueprint) {
	if (dc.Blueps[i]).Compare(blp) <= 0 {
		return
	}
	for i++ ; i < len(dc.Blueps); i++ {
		switch (dc.Blueps[i]).Compare(blp) {
		case 1, 0:
			if blp.Compare(dc.Blueps[i]) == 1 {
				//Are equal
				//fmt.Println("Blueprints equal, return")
				return
			}
			continue
		case -1:
			dc.insert(i, blp)
			return
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	dc.insert(i, blp)
	return
}

func (dc *DynaClient) insert(i int, blp *pb.Blueprint) {
	cnf, err := dc.mgr.NewConfiguration(blp.Add, majQuorum(blp),2 * time.Second)
	if err != nil {
		panic("could not get new config")
	}

	dc.Blueps = append(dc.Blueps, blp)
	dc.Confs = append(dc.Confs, cnf)

	for j:= len(dc.Blueps)-1; j>i; j-- {
		dc.Blueps[j] = dc.Blueps[j-1]
		dc.Confs[j] = dc.Confs[j-1]
	} 

	if len(dc.Blueps) != i + 1 {
		dc.Blueps[i] = blp
		dc.Confs[i] = cnf
	}
}

func (dc *DynaClient) WriteValue(val []byte, st *pb.State) *pb.State {
	if val == nil {
		return st
	}
	return &pb.State{Value: val, Timestamp: st.Timestamp + 1, Writer: dc.ID}
}
