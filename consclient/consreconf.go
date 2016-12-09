package consclient

import (
	"errors"
	"time"

	"github.com/golang/glog"
	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
	smc "github.com/relab/smartMerge/smclient"
)

func (cc *ConsClient) Doreconf(cp conf.Provider, prop *pb.Blueprint, regular int, val []byte) (rst *pb.State, cnt int, err error) {
	if glog.V(6) {
		glog.Infof("C%d: Starting reconfiguration\n", cc.Id)
	}

	doconsensus := true
	cur := 0

forconfiguration:
	for i := 0; i < len(cc.Blueps); i++ {
		if i < cur {
			continue
		}

		var next *pb.Blueprint

		switch prop.Compare(cc.Blueps[i]) {
		case 0, -1:
			//There exists a new proposal
			if doconsensus {
				//Need to agree on new proposal
				var cs int
				next, cs, cur, err = cc.getconsensus(cp, i, prop)
				if err != nil {
					return nil, 0, err
				}
				cnt += cs
			} else {
				next = prop
			}
		case 1:
			// No proposal
			if len(cc.Blueps) == i+1 && (cur == i || regular > 0) {
				// We are in the current configuration, do a read, to check for next configurations. No need to recontact.
				// If atomic: Need to read before writing.
				var st *pb.State
				var c int
				st, cur, c, err = cc.Doread(cp, cur, i, nil)
				if err != nil {
					return nil, 0, err
				}
				cnt += c
				if rst.Compare(st) == 1 {
					rst = st
				}

			}
			if i+1 < len(cc.Blueps) {
				next = cc.Blueps[i+1]
			}

		}
		if i < cur {
			continue forconfiguration
		}

		if cc.Blueps[i].LearnedCompare(next) == 1 {

			cnf := cp.WriteC(cc.Blueps[i], nil)

			writeN := new(pb.AWriteNReply)

			for j := 0; cnf != nil; j++ {
				writeN, err = cnf.AWriteN(&pb.WriteN{
					CurC: uint32(cc.Blueps[i].Len()),
					Next: next,
				})
				cnt++

				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedWriteN: %v\n", cc.Id, err)
					// Try again with full configuration.
					cnf = cp.FullC(cc.Blueps[i])
				}

				if err != nil && j == smc.Retry {
					glog.Errorf("C%d: error %v from WriteN after %d retries: ", cc.Id, err, smc.Retry)
					return nil, 0, err
				}

				if err == nil {
					break
				}
			}

			if glog.V(3) {
				glog.Infof("C%d: CWriteN returned.\n", cc.Id)
			}

			cur = cc.HandleNewCur(cur, writeN.Reply.GetCur())

			if rst.Compare(writeN.Reply.GetState()) == 1 {
				rst = writeN.Reply.GetState()
			}

		} else if i > cur || regular > 1 {
			//Establish new cur, or write value in write, atomic read.

			rst = cc.WriteValue(&val, rst)

			cnf := cp.WriteC(cc.Blueps[i], nil)

			var setS *pb.SetStateReply

			for j := 0; ; j++ {
				setS, err = cnf.SetState(&pb.NewState{
					CurC:  uint32(cc.Blueps[i].Len()),
					State: rst,
				})
				cnt++

				if err != nil && j == 0 {
					glog.Errorf("C%d: error from OptimizedSetState: %v\n", cc.Id, err)
					// Try again with full configuration.
					cnf = cp.FullC(cc.Blueps[i])
				}

				if err != nil && j == smc.Retry {
					glog.Errorf("C%d: error %v from SetState after %d retries: ", cc.Id, err, smc.Retry)
					return nil, 0, err
				}

				if err == nil {
					break
				}
			}

			if i > 0 && glog.V(3) {
				glog.Infof("C%d: Set state in configuration of size %d.\n", cc.Id, cc.Blueps[i].Len())
			} else if glog.V(6) {
				glog.Infof("Set state returned.")
			}

			cur = cc.HandleOneCur(i, setS.Reply.GetCur())
			cc.HandleNext(i, setS.Reply.GetNext())

			if i < len(cc.Blueps)-1 {
				prop = cc.Blueps[len(cc.Blueps)-1]
				doconsensus = false
			}
		}
	}

	cc.SetNewCur(cur)
	if cnt > 2 {
		cc.SetCur(cp, cc.Blueps[0])
		cnt++
	}

	return rst, cnt, nil
}

func (cc *ConsClient) getconsensus(cp conf.Provider, i int, prop *pb.Blueprint) (next *pb.Blueprint, cnt, cur int, err error) {
	ms := 1 * time.Millisecond
	rnd := cc.Id
	// The 24 higher bits of rnd (uint32) are a counter, the lower 8 bits the client id. Should separate the two in the future, to simplify things

prepare:
	for {

		var cnf *pb.Configuration
		//Default leader need not do prepare phase.
		if rnd != 0 {
			//Send Prepare:
			cnf = cp.ReadC(cc.Blueps[i], nil)

			var promise *pb.GetPromiseReply

			for j := 0; ; j++ {
				promise, err = cnf.GetPromise(&pb.Prepare{
					CurC: uint32(cc.Blueps[i].Len()),
					Rnd:  rnd})
				if err != nil && j == 0 {
					glog.Errorf("C%d: error from Optimized Prepare: %v\n", cc.Id, err)
					//Try again with full configuration.
					cnf = cp.FullC(cc.Blueps[i])
				}
				cnt++

				if err != nil && j == smc.Retry {
					glog.Errorf("C%d: error %v from Prepare after %d retries.\n", cc.Id, err, smc.Retry)
					return nil, 0, 0, err
				}

				if err == nil {
					break
				}
			}

			cur = cc.HandleOneCur(i, promise.Reply.GetCur())
			if i < cur {
				glog.V(3).Infof("C%d: Prepare returned new current conf.\n", cc.Id)
				return nil, cnt, cur, nil
			}

			rrnd := promise.Reply.Rnd
			switch {
			case promise.Reply.GetDec() != nil:
				next = promise.Reply.GetDec()
				if glog.V(3) {
					glog.Infof("C%d: Promise reported decided value.\n", cc.Id)
				}
				return
			case rrnd <= rnd:
				// Find the right value to propose, then procede to Accept.
				if promise.Reply.GetVal() != nil {
					next = promise.Reply.Val.Val
					if glog.V(3) {
						glog.Infof("C%d: Re-propose a value.\n", cc.Id)
					}
				} else {
					if glog.V(3) {
						glog.Infof("C%d: Proposing my value.\n", cc.Id)
					}
					next = prop.Merge(cc.Blueps[i]) // This could have side effects on prop. Is this a problem?
					if len(prop.Ids()) == 0 {
						glog.Errorf("Aborting Reconfiguration to avoid unacceptable configuration.")
						return nil, cnt, cur, errors.New("Abort before proposing unacceptable configuration.")
					}
				}
			case rrnd > rnd:
				// Increment round, sleep then return to prepare.
				if glog.V(3) {
					glog.Infof("C%d: Conflict, sleeping %v.\n", cc.Id, ms)
				}

				//The below would be more clear, if we either implement it with a bitshift, or separate the counter, and the id.
				if rrid := rrnd % 256; rrid < cc.Id {
					rnd = rrnd - rrid + cc.Id
				} else {
					rnd = rrnd - rrid + 256 + cc.Id
				}
				time.Sleep(ms)
				ms = 2 * ms
				continue prepare

			}
		} else {
			next = prop.Merge(cc.Blueps[i])
		}

		cnf = cp.WriteC(cc.Blueps[i], nil)

		var learn *pb.AcceptReply

		for j := 0; ; j++ {
			learn, err = cnf.Accept(&pb.Propose{
				CurC: uint32(cc.Blueps[i].Len()),
				Val:  &pb.CV{rnd, next},
			})
			cnt++
			if err != nil && j == 0 {
				glog.Errorf("C%d: error from OptimizedAccept: %v\n", cc.Id, err)
				// Try again with full configuration.
				cnf = cp.FullC(cc.Blueps[i])
			}

			if err != nil && j == smc.Retry {
				glog.Errorf("C%d: error %v from Accept after %d retries: ", cc.Id, err, smc.Retry)
				return nil, 0, cur, err
			}

			if err == nil {
				break
			}
		}

		cur = cc.HandleOneCur(cur, learn.Reply.GetCur())
		if i < cur {
			glog.V(3).Infof("C%d: Accept returned new current conf.\n", cc.Id)
			return
		}

		if learn.Reply.GetDec() == nil && !learn.Reply.Learned {
			if glog.V(3) {
				glog.Infof("C%d: Did not learn, redo prepare.\n", cc.Id)
			}
			rnd += 256
			continue prepare
		}

		if learn.Reply.GetDec() != nil {
			next = learn.Reply.GetDec()
		}

		glog.V(4).Infof("C%d: Did Learn a value.", cc.Id)
		return
	}
}
