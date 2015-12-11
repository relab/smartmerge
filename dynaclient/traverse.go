package dynaclient

import (
	"github.com/golang/glog"

	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
	sm "github.com/relab/smartMerge/smclient"
)

func (dc *DynaClient) Traverse(cp conf.Provider, prop *pb.Blueprint, val []byte, regular bool) (rval []byte, cnt int, err error) {
	rst := new(pb.State)
	for i := 0; i < len(dc.Blueps); i++ {
		var curprop *pb.Blueprint // The current proposal
		if prop != nil && !prop.Equals(dc.Blueps[i]) {
			//Update Snapshot

			cnf := cp.SingleC(dc.Blueps[i])

			getOne := new(pb.GetOneNReply)

			for j := 0; ; j++ {
				getOne, err = cnf.GetOneN(&pb.GetOne{
					Conf: &pb.Conf{
						Cur:  uint32(dc.Blueps[0].Len()),
						This: dc.Confs[i].GlobalID(),
					},
					Next: prop,
				})

				cnt++

				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedGetOne: %v\n", dc.ID, err)
					// Try again with full configuration.
					cnf = dc.Confs[i]
				}

				if err != nil && j == sm.Retry {
					glog.Errorf("C%d: error %v from WriteN after %d retries: ", dc.ID, err, sm.Retry)
					return nil, 0, err
				}

				if err == nil {
					break
				}
			}

			if glog.V(4) {
				glog.Infof("C%d: GetOne returned.\n", dc.ID)
			}

			isnew := dc.handleNewCur(i, getOne.Reply.GetCur(), cp)
			if isnew {
				prop = prop.Merge(getOne.Reply.GetCur())
				i = -1
				continue
			}

			curprop = getOne.Reply.GetNext()

		}

		//Update Snapshot and ReadInView:
		var cnf *pb.Configuration
		if curprop == nil {
			cnf = cp.ReadC(dc.Blueps[i], nil)
		} else {
			cnf = cp.WriteC(dc.Blueps[i], nil)
		}
		writeN := new(pb.DWriteNReply)

		for j := 0; ; j++ {
			writeN, err = dc.Confs[i].DWriteN(
				&pb.DRead{
					Conf: &pb.Conf{
						Cur:  uint32(dc.Blueps[0].Len()),
						This: dc.Confs[i].GlobalID(),
					},
					Prop: curprop,
				})
			cnt++

			if err != nil && j == 0 {
				glog.Errorf("C%d: error from OptimizedWriteN: %v\n", dc.ID, err)
				// Try again with full configuration.
				cnf = dc.Confs[i]
			}

			if err != nil && j == sm.Retry {
				glog.Errorf("C%d: error %v from WriteN after %d retries: ", dc.ID, err, sm.Retry)
				return nil, 0, err
			}

			if err == nil {
				break
			}
		}

		if i > 0 && glog.V(3) {
			glog.Infof("C%d: Read in View with length %d and id %d.\n ", dc.ID, dc.Blueps[i].Len(), dc.Confs[i].GlobalID())
		} else if glog.V(6) {
			glog.Infof("C%d: Read returned.\n", dc.ID)
		}

		isnew := dc.handleNewCur(i, writeN.Reply.GetCur(), cp)
		if isnew {
			if prop != nil {
				prop = prop.Merge(writeN.Reply.GetCur())
			}
			i = -1
			continue
		}

		next := writeN.Reply.GetNext()
		prop = dc.handleNext(i, next, prop, cp)
		if rst.Compare(writeN.Reply.GetState()) == 1 {
			rst = writeN.Reply.GetState()
		}

		if len(next) == 0 && (!regular || i > 0) {

			//WriteInView
			wst := dc.WriteValue(val, rst)

			cnf = cp.WriteC(dc.Blueps[i], nil)

			var setS *pb.DSetStateReply

			for j := 0; ; j++ {
				setS, err = cnf.DSetState(&pb.DNewState{
					Conf: &pb.Conf{
						Cur:  uint32(dc.Blueps[i].Len()),
						This: dc.Confs[i].GlobalID(),
					},
					State: wst,
				})
				cnt++

				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedSetState: %v\n", dc.ID, err)
					// Try again with full configuration.
					cnf = dc.Confs[i]
				}

				if err != nil && j == sm.Retry {
					glog.Errorf("C%d: error %v from SetState after %d retries: ", dc.ID, err, sm.Retry)
					return nil, 0, err
				}

				if err == nil {
					break
				}
			}

			if i > 0 && glog.V(3) {
				glog.Infof("C%d: Write in view with length %d and id %d\n ", dc.ID, dc.Blueps[i].Len(), dc.Confs[i].GlobalID())
			} else if glog.V(6) {
				glog.Infoln("Write returned.")
			}

			isnew = dc.handleNewCur(i, setS.Reply.GetCur(), cp)
			if isnew {
				if prop != nil {
					prop = prop.Merge(setS.Reply.GetCur())
				}
				i = -1
				continue
			}

			dc.Blueps = dc.Blueps[i:]
			dc.Confs = dc.Confs[i:]
			i = 0

			next = setS.Reply.GetNext()
			prop = dc.handleNext(i, next, prop, cp)
		}

		if len(next) > 0 { //Oups this is not just an else to the if above, but can also be used be true, after the WriteInView was executed.
			if len(next) > 1 {
				glog.Errorf("Did not expect ever to receive %d next values with length: %d and %d.\n", len(next), next[0].Len(), next[1].Len())
			}

			regular = false

			cnf = cp.WriteC(dc.Blueps[i], nil)

			var writeNs *pb.DWriteNSetReply

			for j := 0; ; j++ {
				writeNs, err = cnf.DWriteNSet(&pb.DWriteNs{
					Conf: &pb.Conf{
						Cur:  uint32(dc.Blueps[0].Len()),
						This: dc.Confs[i].GlobalID(),
					},
					Next: next,
				})
				cnt++

				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedWriteNSet: %v\n", dc.ID, err)
					// Try again with full configuration.
					cnf = dc.Confs[i]
				}

				if err != nil && j == sm.Retry {
					glog.Errorf("C%d: error %v from WriteNSet after %d retries.\n ", dc.ID, err, sm.Retry)
					return nil, 0, err
				}

				if err == nil {
					break
				}
			}

			if glog.V(3) {
				glog.Infof("C%d: WriteNSet returned.\n", dc.ID)
				if writeNs.Reply.GetCur() != nil {
					glog.Infof("C%d: WriteNSet did return new current.\n", dc.ID)
				}
			}

			isnew = dc.handleNewCur(i, writeNs.Reply.GetCur(), cp)
			if isnew {
				if prop != nil {
					prop = prop.Merge(writeNs.Reply.GetCur())
				}
				i = -1
				continue
			}
			next = writeNs.Reply.GetNext()
			prop = dc.handleNext(i, next, prop, cp)
			continue
		}
	}

	if cnt > 2 {
		dc.SetCur(cp, dc.Blueps[0])
	}

	if val == nil {
		return rst.Value, cnt, nil
	}
	return nil, cnt, nil
}

func (dc *DynaClient) handleNewCur(i int, newCur *pb.Blueprint, cp conf.Provider) bool {
	if newCur == nil {
		return false
	}
	if newCur.Compare(dc.Blueps[i]) == 1 {
		return false
	}

	glog.V(4).Infof("C%d: Found new current view with length\n", dc.ID, newCur.Len())

	cnf := cp.FullC(newCur)

	dc.Blueps = make([]*pb.Blueprint, 1, 5)
	dc.Confs = make([]*pb.Configuration, 1, 5)
	dc.Blueps[0] = newCur
	dc.Confs[0] = cnf

	return true

}

func (dc *DynaClient) handleNext(i int, next []*pb.Blueprint, prop *pb.Blueprint, cp conf.Provider) *pb.Blueprint {
	for _, nxt := range next {
		if nxt != nil {
			dc.findorinsert(i, nxt, cp)
			prop = prop.Merge(nxt)
		}
	}
	return prop
}

func (dc *DynaClient) findorinsert(i int, blp *pb.Blueprint, cp conf.Provider) {
	if (dc.Blueps[i]).Compare(blp) <= 0 {
		return
	}
	for i++; i < len(dc.Blueps); i++ {
		switch (dc.Blueps[i]).Compare(blp) {
		case 1, 0:
			if blp.Compare(dc.Blueps[i]) == 1 {
				//Are equal
				return
			}
			continue
		case -1:
			dc.insert(i, blp, cp)
			return
		}
	}
	//fmt.Println("Inserting new highest blueprint")
	dc.insert(i, blp, cp)
	return
}

func (dc *DynaClient) insert(i int, blp *pb.Blueprint, cp conf.Provider) {
	glog.V(4).Infof("C%d: Found next blueprint.\n", dc.ID)

	cnf := cp.FullC(blp)

	dc.Blueps = append(dc.Blueps, blp)
	dc.Confs = append(dc.Confs, cnf)

	for j := len(dc.Blueps) - 1; j > i; j-- {
		dc.Blueps[j] = dc.Blueps[j-1]
		dc.Confs[j] = dc.Confs[j-1]
	}

	if len(dc.Blueps) != i+1 {
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

func (dc *DynaClient) SetCur(cp conf.Provider, cur *pb.Blueprint) {
	cnf := cp.WriteC(cur, nil)

	for j := 0; ; j++ {
		_, err := cnf.DSetCur(&pb.NewCur{
			CurC: uint32(cur.Len()),
			Cur:  cur})

		if err != nil && j == 0 {
			glog.Errorf("C%d: error from Thrifty New Cur: %v\n", dc.ID, err)
			// Try again with full configuration.
			cnf = cp.FullC(cur)
		}

		if err != nil && j == sm.Retry {
			glog.Errorf("C%d: error %v from NewCur after %d retries: ", dc.ID, err, sm.Retry)
			break
		}

		if err == nil {
			break
		}
	}
}
