package smclient

import (
	"errors"
	"time"

	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

func (smc ConfigProvider) consreconf(prop *pb.Blueprint, regular bool, val []byte) (rst *pb.State, cnt int, err error) {
	if glog.V(6) {
		glog.Infof("C%d: Starting reconfiguration\n", smc.ID)
	}

	doconsensus := true
	cur := 0
	var rid []uint32
	
forconfiguration:
	for i := 0; i < smc.getNBlueps(); i++ {
		if i < cur {
			continue
		}

		var next *pb.Blueprint

		switch prop.Compare(smc.getBluep(i)) {
		case 0, -1:
			//There exists a new proposal
			if doconsensus {
				//Need to agree on new proposal
				var cs int
				next, cs, cur, err = smc.getconsensus(i, prop)
				if err != nil {
					return nil, 0, err
				}
				cnt += cs
			} else {
				next = prop
			}
		case 1:
			// No proposal
			var st *pb.State
			st, cur, err = smc.doread(cur, i)
			if err != nil {
				return nil, 0, err
			}
			if i+1 < smc.getNBlueps() {
				next = smc.getBluep(i+1)
			}
			cnt++
			if rst.Compare(st) == 1 {
				rst = st
			}
		}
		if i < cur {
			continue forconfiguration
		}

		if smc.getBluep(i).LearnedCompare(next) == 1 {
			
			cnf := smc.getWriteC(i, nil)

			writeN := new(pb.AWriteNReply)

			for j := 0; cnf != nil; j++ {
				writeN, err = cnf.AWriteN(&pb.WriteN{
						CurC: uint32(smc.getLenBluep(i)), 
						Next: next,
					})
				cnt++

				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedWriteN: %v\n", smc.getId(), err)
					// Try again with full configuration.
					cnf = smc.getFullC(i)
				}

				if err != nil && j == Retry {
					glog.Errorf("C%d: error %v from WriteN after %d retries: ", smc.getId(), err, Retry)
					return nil, 0, err
				}

				if err == nil {
					break
				}
			}
			
			
			if glog.V(3) {
				glog.Infof("C%d: CWriteN returned.\n", smc.getId())
			}

			cur = smc.handleNewCur(cur, writeN.Reply.GetCur())

			if rst.Compare(writeN.Reply.GetState()) == 1 {
				rst = writeN.Reply.GetState()
			}
			
			if c := writeN.Reply.GetCur(); c == nil || !c.Abort {
				rid = pb.Union(rid, writeN.MachineIDs)
			}

		} else if i > cur || ! regular {
			//Establish new cur, or write value in write, atomic read.
	
			rst = smc.WriteValue(val, rst)
	
			cnf := smc.getWriteC(i, nil)

			var setS *pb.SetStateReply

			for j := 0; ; j++ {
				setS, err = cnf.SetState(&pb.NewState{
					CurC: uint32(smc.getLenBluep(i)),
					Cur: smc.getBluep(i),
					State: rst,
				})
				cnt++

				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedSetState: %v\n", smc.getId(), err)
					// Try again with full configuration.
					cnf = smc.getFullC(i)
				}

				if err != nil && j == Retry {
					glog.Errorf("C%d: error %v from SetState after %d retries: ", smc.getId(), err, Retry)
					return nil, 0, err
				}

				if err == nil {
					break
				}
			}
	
			if i > 0 && glog.V(3) {
				glog.Infof("C%d: Set state in configuration of size %d.\n", smc.ID, smc.Blueps[i].Len())
			} else if glog.V(6) {
				glog.Infof("Set state returned.")
			}
	
			cur = smc.handleOneCur(i, setS.Reply.GetCur())
			smc.handleNext(i, setS.Reply.GetNext())
	
			if !regular && i < len(smc.Confs)-1 {
				prop = smc.Blueps[len(smc.Blueps)-1]
				doconsensus = false
				goto forconfiguration
			}
		}
	}
	
	smc.Blueps = smc.Blueps[cur:]
	smc.Confs = smc.Confs[cur:]

	return rst, cnt, nil
}

func (smc *SmClient) getconsensus(i int, prop *pb.Blueprint) (next *pb.Blueprint, cnt, cur int, err error) {
	ms := 1 * time.Millisecond
	rnd := smc.ID
prepare:
	for {
		//Send Prepare:
		promise, errx := smc.Confs[i].GetPromise(&pb.Prepare{CurC: uint32(smc.Blueps[i].Len()), Rnd: rnd})
		if errx != nil {
			//Should log this for debugging
			glog.Errorf("C%d: Prepare returned error: %v\n", smc.ID, errx)
			return nil, 0, i, errx
		}
		cnt++
		cur = smc.handleOneCur(i, promise.Reply.GetCur(), true)
		if i < cur {
			glog.V(3).Infof("C%d: Prepare returned new current conf.\n", smc.ID)
			return nil, cnt, cur, nil
		}

		rrnd := promise.Reply.Rnd
		switch {
		case promise.Reply.GetDec() != nil:
			next = promise.Reply.GetDec()
			if glog.V(3) {
				glog.Infof("C%d: Promise reported decided value.\n", smc.ID)
			}
			return
		case rrnd <= rnd:
			if promise.Reply.GetVal() != nil {
				next = promise.Reply.Val.Val
				if glog.V(3) {
					glog.Infof("C%d: Re-propose a value.\n", smc.ID)
				}
			} else {
				if glog.V(3) {
					glog.Infof("C%d: Proposing my value.\n", smc.ID)
				}
				if len(prop.Ids()) == 0 {
					glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
					return nil, cnt, cur, errors.New("Abort before proposing unacceptable configuration.")
				}
				next = prop.Merge(smc.Blueps[i])
			}
		case rrnd > rnd:
			// Increment round, sleep then return to prepare.
			if glog.V(3) {
				glog.Infof("C%d: Conflict, sleeping %v.\n", smc.ID, ms)
			}
			if rrid := rrnd % 256; rrid < smc.ID {
				rnd = rrnd - rrid + smc.ID
			} else {
				rnd = rrnd - rrid + 256 + smc.ID
			}
			time.Sleep(ms)
			ms = 2 * ms
			continue prepare

		}

		learn, errx := smc.Confs[i].Accept(&pb.Propose{CurC: uint32(smc.Blueps[i].Len()), Val: &pb.CV{rnd, next}})
		if err != nil {
			glog.Errorf("C%d: Accept returned error: %v\n", smc.ID, errx)
			return nil, 0, cur, errx
		}

		cnt++
		cur = smc.handleOneCur(cur, learn.Reply.GetCur(), true)
		if i < cur {
			glog.V(3).Infof("C%d: Accept returned new current conf.\n", smc.ID)
			return
		}

		if learn.Reply.GetDec() == nil && !learn.Reply.Learned {
			if glog.V(3) {
				glog.Infof("C%d: Did not learn, redo prepare.\n", smc.ID)
			}
			rnd += 256
			continue prepare
		}

		if learn.Reply.GetDec() != nil {
			next = learn.Reply.GetDec()
		}

		glog.V(4).Infof("C%d: Did Learn a value.", smc.ID)
		return
	}
}
