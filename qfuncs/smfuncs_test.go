package qfuncs

import (
	"testing"

	pr "github.com/relab/smartMerge/proto"
)

var qspec pr.QuorumSpec

func TestImplements(t *testing.T) {
	smqs := SMQuorumSpec{}
	qspec = &smqs
}
