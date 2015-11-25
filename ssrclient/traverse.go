package ssrclient

import (
	//"errors"

	"github.com/golang/glog"
	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
	smc "github.com/relab/smartMerge/smclient"
)

func (ssc *SSRClient) Doreconf(cp conf.Provider, prop *pb.Blueprint, regular bool, val []byte) (rst *pb.State, cnt int, err error) {
	if glog.V(6) {
		glog.Infof("C%d: Starting doreconfiguration\n", ssc.Id)
	}

	for i := 0; i < len(ssc.Blueps); i++ {

		if i == 0 && prop != nil {
			prop = prop.Merge(ssc.Blueps[0])
		}

		var c int
		var newcur bool

		prop, c, newcur, err = ssc.spsn(cp, i, prop)
		if err != nil {
			return nil, 0, err
		}

		cnt += c

		if newcur {
			// Restart in the new current configuration.
			i = -1
			continue
		}

		cnf := cp.ReadC(ssc.Blueps[i], nil)

		// Maybe we can omit this for reconfigurations in the last configuration.
		readS := new(pb.SReadSReply)
		for j := 0; cnf != nil; j++ {
			readS, err = cnf.SReadS(&pb.SRead{
				CurL: uint32(ssc.Blueps[0].Len()),
			})
			cnt++
			if err == nil {
				break
			}

			if err != nil && j == 0 {
				glog.Errorf("C%d: error from ThriftyReadS: %v\n", ssc.Id, err)
				// Try again with full configuration.
				cnf = cp.FullC(ssc.Blueps[i])
			}
			if err != nil && j == smc.Retry {
				glog.Errorf("C%d: error %v from ReadS after %d retries: ", ssc.Id, err, smc.Retry)
				return nil, 0, err
			}
		}
		if glog.V(6) {
			glog.Infof("C%d: ReadS returned.\n", ssc.Id)
		}

		if cr := readS.Reply.Cur; cr != nil {
			ssc.Blueps = []*pb.Blueprint{cr}
			glog.V(3).Infof("C%d: ReadS returned new current conf of length %d.\n", ssc.Id, cr.Len())
			i = -1
			continue
		}

		if rst.Compare(readS.Reply.GetState()) == 1 {
			rst = readS.Reply.GetState()
		}

		if i > 0 || !regular {
			//Establish new cur, or write value in write, atomic read.

			rst = ssc.WriteValue(&val, rst)

			cnf = cp.WriteC(ssc.Blueps[i], nil)

			var setS *pb.SSetStateReply

			for j := 0; ; j++ {
				setS, err = cnf.SSetState(&pb.SState{
					CurL:  uint32(ssc.Blueps[i].Len()),
					Cur:   ssc.Blueps[i],
					State: rst,
				})
				cnt++

				if err == nil {
					break
				}

				if err != nil && j == 0 {
					glog.Errorf("C%d: error from Thrifty SetState: %v\n", ssc.Id, err)
					// Try again with full configuration.
					cnf = cp.FullC(ssc.Blueps[i])
				}

				if err != nil && j == smc.Retry {
					glog.Errorf("C%d: error %v from SetState after %d retries: ", ssc.Id, err, smc.Retry)
					return nil, 0, err
				}
			}

			if i > 0 && glog.V(3) {
				glog.Infof("C%d: Set state in configuration of size %d.\n", ssc.Id, ssc.Blueps[i].Len())
			} else if glog.V(6) {
				glog.Infof("Set state returned.")
			}

			if regular {
				ssc.Blueps = []*pb.Blueprint{ssc.Blueps[i]}
				return
			}

			if cr := setS.Reply.Cur; cr != nil {
				ssc.Blueps = []*pb.Blueprint{cr}
				glog.V(3).Infof("C%d: SetState returned new current conf of length %d.\n", ssc.Id, cr.Len())
				i = -1
				continue
			}

			if setS.Reply.HasNext {
				glog.V(4).Infoln("There is a next configuration. Restart.")
				i--
				continue
			}
		}
	}

	return rst, cnt, nil
}

func (ssc *SSRClient) spsn(cp conf.Provider, i int, prop *pb.Blueprint) (next *pb.Blueprint, cnt int, cur bool, err error) {

	for rnd := 0; ; rnd++ {
		//Do SpSn Phase 1:
		cnf := cp.WriteC(ssc.Blueps[i], nil)

		var collect *pb.SpSnOneReply

		for j := 0; ; j++ {
			collect, err = cnf.SpSnOne(&pb.SWriteN{
				CurL: uint32(ssc.Blueps[0].Len()),
				This: uint32(ssc.Blueps[i].Len()),
				Rnd:  uint32(rnd),
				Prop: prop,
			})
			if err != nil && j == 0 {
				glog.Errorf("C%d: error from OptimizedSpSnOne: %v\n", ssc.Id, err)
				//Try again with full configuration.
				cnf = cp.FullC(ssc.Blueps[i])
			}
			cnt++

			if err != nil && j == smc.Retry {
				glog.Errorf("C%d: error %v from Phase1 after %d retries.\n", ssc.Id, err, smc.Retry)
				return nil, 0, false, err
			}

			if err == nil {
				break
			}
		}

		// Abort on new Cur
		if cr := collect.Reply.Cur; cr != nil {
			ssc.Blueps = []*pb.Blueprint{cr}
			glog.V(3).Infof("C%d: Phase1 returned new current conf of length %d.\n", ssc.Id, cr.Len())
			return prop, cnt, true, nil
		}

		// Merge with other proposals, or commit.
		commit := true
		for i, blp := range collect.Reply.Next {
			if i == 0 && blp.Equals(prop) {
				continue
			}
			commit = false
			prop = prop.Merge(blp)
		}

		if prop.Len() == 0 && commit {
			if glog.V(6) {
				glog.Infof("C%d: Empty Phase1 returned commit.\n", ssc.Id)
			}
			return nil, cnt, false, nil
		}

		if commit {
			ssc.HandleOneCur(i, prop)
		}

		if glog.V(4) {
			glog.Infof("C%d: Phase1 returned in rnd %d, commit is %v\n", ssc.Id, rnd, commit)
		}

		//Do SpSn Phase two.
		cnf = cp.WriteC(ssc.Blueps[i], nil) //This is not really necessary.

		var commitR *pb.SCommitReply

		for j := 0; ; j++ {
			commitR, err = cnf.SCommit(&pb.Commit{
				CurL:    uint32(ssc.Blueps[0].Len()),
				This:    uint32(ssc.Blueps[i].Len()),
				Rnd:     uint32(rnd),
				Commit:  commit,
				Collect: prop,
			})
			cnt++

			if err != nil && j == 0 {
				glog.Errorf("C%d: error from OptimizedCommit: %v\n", ssc.Id, err)
				// Try again with full configuration.
				cnf = cp.FullC(ssc.Blueps[i])
			}

			if err != nil && j == smc.Retry {
				glog.Errorf("C%d: error %v from Commit after %d retries: ", ssc.Id, err, smc.Retry)
				return nil, 0, false, err
			}

			if err == nil {
				break
			}
		}

		// Abort on new Cur.
		if cr := commitR.Reply.Cur; cr != nil {
			ssc.Blueps = []*pb.Blueprint{cr}
			glog.V(3).Infof("C%d: Commit returned new current conf of length %d.\n", ssc.Id, cr.Len())
			return prop, cnt, true, nil
		}

		//Insert Committed Blueprint
		if commit {
			ssc.HandleOneCur(i, prop)
		} else {
			ssc.HandleOneCur(i, commitR.Reply.Committed)
		}

		//If no uncommitted was collected, return.
		if commitR.Reply.Collected.Len() == 0 {
			if glog.V(4) {
				glog.Infof("C%d: Commit returned in rnd %d, nothing collected.", ssc.Id, rnd)
			}
			return prop, cnt, false, nil
		}

		if glog.V(4) {
			glog.Infof("C%d: Commit returned in rnd %d. Length collected is %d.\n", ssc.Id, rnd, commitR.Reply.Collected.Len())
		}

		//Merge with collected and go to next rnd.
		prop = prop.Merge(commitR.Reply.Collected)
	}
}
