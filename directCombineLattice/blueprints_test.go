package directCombineLattice

import (
	"testing"
	//"fmt"
)

var a1 = ProcSet(map[ID]bool{ID(1):true, ID(2):true})
var a2 = ProcSet(map[ID]bool{ID(1):true, ID(3):true})
var r1 = ProcSet(map[ID]bool{ID(1):true})
var r2 = ProcSet(map[ID]bool{ID(3):true})

func TestUnion(t *testing.T) {
	u1 := r1.union(r2)
	for id := range u1 {
		if !id == ID(1) && !id == ID(3) {
			t.Error("Union did introduce new element.")
		}
	}
	if len(u1) != 2 {
		t.Error("Union does not have the right number of elements.")
	}
}