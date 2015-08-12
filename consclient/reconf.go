package consclient

import (
	"errors"
	"fmt"
	"time"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
)

func (cc *CClient) Reconf(prop *lat.Blueprint) (cnt int, err error) {
	//Proposed blueprint is already in place, or outdated.
	if prop.Compare(cc.Blueps[0]) == 1 {
		return cnt, nil
	}

	if len(prop.Ids()) == 0 {
		return cnt, errors.New("Abort before proposing unacceptable configuration.")
	}

	cur := 0
	
	var (
		rrnd uint32
		next *lat.Blueprint
		dec bool
		backup bool
		newCur *lat.Blueprint
		err error
	)
	rst := new(pb.State)
	rnd := cc.ID
	forloop:
	for i := 0; i < len(cc.Confs); i++ {
		if i < cur {
			continue
		}
		
		if cc.Blueps[i].Equals(prop) {
			next = nil
			goto decide
		}
			
		
		ms := 1 * time.Millisecond
		prepare:
		rrnd, dec, backup, next, newCur, err = cc.Confs[i].CPrepare(cc.Blueps[i], rnd)
		cnt++
		cur = cc.handleNewCur(cur, newCur)
		if i < cur {
			continue
		}
		
		if err != nil {
			//Should log this for debugging
			fmt.Println("Prepare returned error: ",err)
			panic("Error from CPrepare")
		}

		switch {
		case dec:
			goto decide
		case rrnd <= rnd && next == nil:
			prop = prop.Merge(cc.Blueps[i])
			next = prop
		case rrnd > rnd && !backup:
			rnd = rrnd
		case rrnd > rnd && backup: 
			if rrid := rrnd%256; rrid < cc.ID {
				rnd = rrnd-rrid+cc.ID
			} else {
				rnd = rrnd-rrid+256+cc.ID
			}
			time.Sleep(ms)
			ms = 2*ms
			goto prepare
		}
		
	
		next, dec, newCur, err = cc.Confs[i].CAccept(cc.Blueps[i],rnd, next)
		cnt++
		cur = cc.handleNewCur(cur, newCur)
		if i < cur {
			continue
		}
		
		if err != nil {
			//Should log this for debugging
			fmt.Println("Accept returned error: ",err)
			panic("Error from CAccept")
		}

		
		if next == nil && !dec {
			goto prepare
		}
		
		decide:
		st, _, newCur, err := cc.Confs[i].CReadS(cc.Blueps[i], cc.Confs[i].ID(), next)
		cnt++
		cur = cc.handleNewCur(cur, newCur)
		if err != nil && cur <= i {
			fmt.Println("error from AReadS: ", err)
			//No Quorum Available. Retry
			panic("Aread returned error")
		}
		cc.handleNext(i, next)
		
		if rst.Compare(st) == 1 {
			rst = st
		}
	}

	if i := len(cc.Confs) - 1; i > cur {
		err := cc.Confs[i].CSetState(cc.Blueps[i], rst)
		cnt++
		if err != nil {
			//Not sure what to do:
			fmt.Println("SetState returned error, not sure what to do")
			panic("Error from SetState")
		}
		cur = i
	}

	cc.Blueps = cc.Blueps[cur:]
	cc.Confs = cc.Confs[cur:]

	return cnt, nil
}
