package qfuncs

import (
	"fmt"
	"testing"

	bp "github.com/relab/smartMerge/blueprints"
	pr "github.com/relab/smartMerge/proto"
)

var qspec pr.QuorumSpec

var one = uint32(2)
var n11 = &bp.Node{Id: one, Version: one}
var b1 = &bp.Blueprint{Nodes: []*bp.Node{n11}, FaultTolerance: one, Epoch: one}

func TestImplements(t *testing.T) {
	smqs := SMQuorumSpec{}
	qspec = &smqs
}

func TestQspecFromBP(t *testing.T) {
	qs := SMQSpecFromBP(b1)
	fmt.Printf("%#v\n", qs)
	fmt.Printf("b1.Quorum %d, b1.Size %d", b1.Quorum(), b1.Size())
}
