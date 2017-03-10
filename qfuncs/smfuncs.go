package qfuncs

import (
	"github.com/golang/glog"
	bp "github.com/relab/smartmerge/blueprints"
	pr "github.com/relab/smartmerge/proto"
)

type SMQuorumSpec struct {
	q int //quorum size
	n int //configuration size

	rq  int //read-quorum size
	wq  int //write-quorum size
	rwq int //size including both read and write quorum
}

func NewSMQSpec(q, n int) *SMQuorumSpec {
	return &SMQuorumSpec{
		q:   q,
		n:   n,
		rq:  ReadQuorum(q, n),
		wq:  WriteQuorum(q, n),
		rwq: MaxQuorum(q, n),
	}
}

// ReadQuorum returns the size of a read quorum.
// If q is larger than half of the nodes in the configuration (rounded up),
// The ReadQuorum size will be smaller.
// Otherwise, a ReadQuorum includes half of the nodes (rounded up).
func ReadQuorum(q, n int) int {
	return n - q + 1
}

// WriteQuorum returns the size of a WriteQuorum.
// WriteQuorum may be larger than ReadQuorums.
func WriteQuorum(q, n int) int {
	return q
}

// MaxQuorum returns the size of a Quorum
// that is both a read and write quorum.
func MaxQuorum(q, n int) int {
	if WriteQuorum(q, n) > ReadQuorum(q, n) {
		return WriteQuorum(q, n)
	}
	return ReadQuorum(q, n)
}

func SMQSpecFromBP(b *bp.Blueprint) *SMQuorumSpec {
	return NewSMQSpec(b.Quorum(), b.NSize())
}

// ConfResponder is an interface that wraps all messages that return a ConfReply.
// These are ReadReply, WriteNReply, and LAReply
type ConfResponder interface {
	GetCur() *pr.ConfReply
}

func checkConfResponder(cr ConfResponder) bool {
	return checkConfReply(cr.GetCur())
}

func checkConfReply(cr *pr.ConfReply) bool {
	if cr != nil && cr.Abort {
		return true
	}
	return false
}

func handleConfResponder(old *pr.ConfReply, cr ConfResponder) *pr.ConfReply {
	return handleConfReply(old, cr.GetCur())
}

func handleConfReply(old *pr.ConfReply, cr *pr.ConfReply) *pr.ConfReply {
	if old == nil {
		return cr
	}

	if cr == nil {
		return old
	}
	if old.Cur.LearnedCompare(cr.Cur) == 1 {
		old.Cur = cr.Cur
	}

	old.Next = GetBlueprintSlice(old.Next, cr)
	return old
}

func (qs *SMQuorumSpec) FwdQF(replies []*pr.Ack) (*pr.Ack, bool) {
	if len(replies) < qs.rq {
		return nil, false
	}
	return replies[0], true
}

func (qs *SMQuorumSpec) ReadQF(replies []*pr.ReadReply) (*pr.ReadReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if checkConfResponder(lastrep) {
		if glog.V(3) {
			glog.Infoln("ReadS reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	if len(replies) < qs.rq {
		if glog.V(7) {
			glog.Infoln("Not enough ReadSReplies yet.")
		}
		return nil, false
	}

	lastrep = new(pr.ReadReply)
	for _, rep := range replies {
		if lastrep.GetState().Compare(rep.GetState()) == 1 {
			lastrep.State = rep.GetState()
		}
		lastrep.Cur = handleConfResponder(lastrep.Cur, rep) // I think the assignment can be omitted.
	}

	return lastrep, true
}

func (qs *SMQuorumSpec) WriteQF(replies []*pr.ConfReply) (*pr.ConfReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if checkConfReply(lastrep) {
		if glog.V(3) {
			glog.Infoln("WriteS reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < qs.rwq {
		if glog.V(7) {
			glog.Infoln("Not enough WriteSReplies yet.")
		}
		return nil, false
	}

	lastrep = new(pr.ConfReply)
	for _, rep := range replies {
		lastrep = handleConfReply(lastrep, rep)
	}

	return lastrep, true
}

func (qs *SMQuorumSpec) WriteNextQF(replies []*pr.WriteNReply) (*pr.WriteNReply, bool) {
	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if checkConfResponder(lastrep) {
		if glog.V(3) {
			glog.Infoln("WriteN reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < qs.rwq {
		return nil, false
	}

	lastrep = new(pr.WriteNReply)
	for _, rep := range replies {
		if lastrep.GetState().Compare(rep.GetState()) == 1 {
			lastrep.State = rep.GetState()
		}
		lastrep.LAState = lastrep.GetLAState().Merge(rep.GetLAState())
		lastrep.Cur = handleConfResponder(lastrep.Cur, rep)
	}

	return lastrep, true
}

func (qs *SMQuorumSpec) SetCurQF(replies []*pr.NewCurReply) (*pr.NewCurReply, bool) {
	// Return false, if not enough replies yet.
	if len(replies) < qs.wq {
		return nil, false
	}

	for _, rep := range replies {
		if rep != nil && !rep.New {
			return rep, true
		}
	}
	return replies[0], true
}

func (qs *SMQuorumSpec) LAPropQF(replies []*pr.LAReply) (*pr.LAReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if checkConfResponder(lastrep) {
		if glog.V(3) {
			glog.Infoln("LAProp reported new Cur.")
		}
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < qs.rwq {
		return nil, false
	}

	lastrep = new(pr.LAReply)
	for _, rep := range replies {
		lastrep.LAState = lastrep.GetLAState().Merge(rep.GetLAState())
		lastrep.Cur = handleConfResponder(lastrep.Cur, rep)
	}

	return lastrep, true
}

func (qs *SMQuorumSpec) SetStateQF(replies []*pr.NewStateReply) (*pr.NewStateReply, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	if len(replies) < qs.rwq {
		return nil, false
	}

	next := make([]*bp.Blueprint, 0, 1)
	for _, rep := range replies {
		next = GetBlueprintSlice(next, rep)
	}

	lastrep.Next = next

	return lastrep, true
}

// NextReport is an interface that wraps all message that include an array of
// next bluerprints. These are: NewStateReply, ConfReply and all messages that
// include a ConfReply, see ConfResponder above.
type NextReport interface {
	GetNext() []*bp.Blueprint
}

func (qs *SMQuorumSpec) GetPromiseQF(replies []*pr.Promise) (*pr.Promise, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < qs.rq {
		return nil, false
	}

	lastrep = new(pr.Promise)
	for _, rep := range replies {
		if rep == nil {
			continue
		}

		if rep.GetDec() != nil {
			return rep, true
		}

		if rep.Rnd > lastrep.Rnd {
			lastrep.Rnd = rep.Rnd
		}
		if rep.Val == nil {
			continue
		}
		if lastrep.Val == nil || rep.Val.Rnd > lastrep.Val.Rnd {
			lastrep.Val = rep.Val
		}
	}

	return lastrep, true
}

func (qs *SMQuorumSpec) AcceptQF(replies []*pr.Learn) (*pr.Learn, bool) {

	// Stop RPC if new current configuration reported.
	lastrep := replies[len(replies)-1]
	if lastrep.GetCur() != nil {
		return lastrep, true
	}

	// Return false, if not enough replies yet.
	// This rpc is both reading and writing.
	if len(replies) < qs.rwq {
		return nil, false
	}

	lastrep = new(pr.Learn)
	lastrep.Learned = true
	for _, rep := range replies {
		if rep == nil || !rep.Learned {
			lastrep.Learned = false
		}

		if rep.GetDec() != nil {
			return rep, true
		}
	}

	return lastrep, true

}

func GetBlueprintSlice(next []*bp.Blueprint, rep NextReport) []*bp.Blueprint {
	repNext := rep.GetNext()
	if repNext == nil {
		return next
	}

	if next == nil {
		next = make([]*bp.Blueprint, 0, len(repNext))
	}
	for _, blp := range repNext {
		next = addLearned(next, blp)
	}

	return next
}

func addLearned(bls []*bp.Blueprint, bp *bp.Blueprint) []*bp.Blueprint {
	place := 0

findplacefor:
	for _, blpr := range bls {
		switch blpr.LearnedCompare(bp) {
		case 0:
			//New blueprint already present
			return bls
		case -1:
			break findplacefor
		default:
			place++
			continue
		}
	}

	bls = append(bls, nil)

	for i := len(bls) - 1; i > place; i-- {
		bls[i] = bls[i-1]
	}
	bls[place] = bp

	return bls
}

type LAStateReport interface {
	GetLAState() *bp.Blueprint
}

func MergeLAState(las *bp.Blueprint, rep LAStateReport) *bp.Blueprint {
	lap := rep.GetLAState()
	if lap == nil {
		return las
	}
	if las == nil {
		return lap
	}
	return las.Merge(lap)
}

type CurReport interface {
	GetCur() *bp.Blueprint
}
