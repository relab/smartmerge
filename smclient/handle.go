package smclient

import (
	"github.com/golang/glog"

	bp "github.com/relab/smartMerge/blueprints"
	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
)

// SetNewCur truncates the list of blueprints.
// The argument is the index of the new first element.
// SetNewCur should be used to remove all earlier configurations
// after installing a new current configuraiton.
func (smc *SmClient) SetNewCur(cur int) {
	if cur >= len(smc.Blueps) {
		glog.Fatalln("Index for new cur out of bound.")
	}

	if cur == 0 {
		return
	}

	smc.Blueps = smc.Blueps[cur:]
}

// HandleOneCur can be used to add a new current configuration
// during a traversal of the list of configurations/blueprints.
// The new blueprint is inserted in the list.
// The method also takes an argument
// cur, indicating the current index in the list.
// If the newly installed blueprint is more uptodate than the bluerprint
// at that index, the index of the newly installed blueprint is returned.
func (smc *SmClient) HandleOneCur(cur int, newCur *bp.Blueprint) int {
	if newCur == nil {
		return cur
	}
	if glog.V(7) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Len(), smc.Blueps[cur].Len())
	}
	return smc.findorinsert(cur, newCur)
}

// HandleNewCur can be used to update the list of blueprints/configurations
// with a new confReply received during a traversal.
// Functions similar to HandleOneCur, but can also update information about
// configurations not yet installed.
func (smc *SmClient) HandleNewCur(cur int, newCur *pb.ConfReply) int {
	if newCur == nil {
		return cur
	}
	smc.HandleNext(cur, newCur.Next)
	if newCur.Cur == nil {
		return cur
	}
	if glog.V(3) {
		glog.Infof("Found new Cur with length %d, current has length %d\n", newCur.Cur.Len(), smc.Blueps[cur].Len())
	}

	return smc.findorinsert(cur, newCur.Cur)
}

// HandleNext inserts several new blueprints into the list.
// Input should be ordered.
func (smc *SmClient) HandleNext(i int, next []*bp.Blueprint) {
	if len(next) == 0 {
		return
	}

	for _, nxt := range next {
		if nxt != nil {
			i = smc.findorinsert(i, nxt)
		}
	}
}

func (smc *SmClient) findorinsert(i int, blp *bp.Blueprint) int {
	old := true
	for ; i < len(smc.Blueps); i++ {
		switch smc.Blueps[i].LearnedCompare(blp) {
		case 0:
			return i
		case 1:
			old = false
			continue
		case -1:
			if old { //This is an outdated blueprint.
				return i
			}
			smc.insert(i, blp)
			return i
		}
	}
	smc.insert(i, blp)
	return i
}

func (smc *SmClient) insert(i int, blp *bp.Blueprint) {
	glog.V(3).Infof("Inserting new blueprint with length %d at place %d\n", blp.Len(), i)

	smc.Blueps = append(smc.Blueps, blp)

	for j := len(smc.Blueps) - 1; j > i; j-- {
		smc.Blueps[j] = smc.Blueps[j-1]
	}

	if len(smc.Blueps) != i+1 {
		smc.Blueps[i] = blp
	}
}

// SetCur informs the servers in the configuration, belonging to cur,
// that this configuration is installed.
func (smc *SmClient) SetCur(cp conf.Provider, cur *bp.Blueprint) {
	cnf := cp.WriteC(cur, nil)

	for j := 0; ; j++ {
		_, err := cnf.SetCur(context.Background(), &pb.NewCur{
			CurC: uint32(cur.Len()),
			Cur:  cur})

		if err != nil && j == 0 {
			glog.Errorf("C%d: error from Thrifty New Cur: %v\n", smc.Id, err)
			// Try again with full configuration.
			cnf = cp.FullC(cur)
		}

		if err != nil && j == Retry {
			glog.Errorf("C%d: error %v from NewCur after %d retries: ", smc.Id, err, Retry)
			break
		}

		if err == nil {
			break
		}
	}
}
