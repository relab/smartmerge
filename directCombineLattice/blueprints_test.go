package directCombineLattice

import (
	"fmt"
	"testing"
)

var one = ID(1)
var two = ID(2)
var tre = ID(3)

var a0 = map[ID]bool{}
var a1 = map[ID]bool{ID(1): true, ID(2): true}
var a2 = map[ID]bool{ID(1): true, ID(3): true}
var r1 = map[ID]bool{ID(1): true}
var r2 = map[ID]bool{ID(3): true}
var a3 = map[ID]bool{ID(2): true, ID(3): true}

func TestUnion(t *testing.T) {
	u1 := union(r1, r2)
	for id := range u1 {
		if (id != one) && id != tre {
			t.Error("Union did introduce new element.")
		}
	}
	if len(u1) != 2 {
		t.Error("Union does not have the right number of elements.")
	}

	u1 = union(a1, a2)

	for id := range u1 {
		if id != one && id != tre && id != two {
			t.Error("Union did introduce new element.")
		}
	}
	if len(u1) != 3 {
		t.Error("Union does not have the right number of elements.")
	}
}

func TestDifference(t *testing.T) {
	df := difference(a2, r2)
	for id := range df {
		if id != one {
			t.Error("Unwanted element in difference.")
		}
	}
	if len(df) != 1 {
		t.Error("Difference does not have the right number of elements.")
	}

	df = difference(a1, a2)

	if len(df) != 1 {
		t.Error("Difference does not have the right number of elements.")
	}
	for id := range df {
		if id != two {
			t.Errorf("Difference(%v,%v) was %v, not ID(2).", a1, a2, df)
		}
	}
}

func TestSubset(t *testing.T) {
	if !subset(r1, a1) {
		t.Errorf("%v was no subset of %v.", r1, a1)
	}
	if subset(a2, a1) {
		t.Errorf("%v was subset of %v.", a2, a1)
	}
}

var bp1 = &Blueprint{r1, r2}
var bp2 = &Blueprint{a1, r2}
var bp3 = &Blueprint{a2, a0}
var bpx = &Blueprint{r2, r1}
var bpy = &Blueprint{a1, a0}
var bpz = &Blueprint{a3, r1}

func TestCompar(t *testing.T) {
	fmt.Printf("Rem: %v, len: %d", bp3.Rem, len(bp3.Rem))
	mybl := new(Blueprint)
	fmt.Printf("Rem: %v, len: %d", mybl.Rem, len(mybl.Rem))
	if bp1.Compare(bp2) != 1 {
		t.Errorf("%v not smaller %v.", bp1, bp2)
	}
	if bp2.Compare(bp1) != -1 {
		t.Errorf("%v not larger %v.", bp2, bp1)
	}
	if bpx.Compare(bpy) != 0 {
		t.Errorf("%v comparable to %v.", bpx, bpy)
	}
	if bp3.Compare(bp1) != 1 {
		t.Errorf("%v not larger %v.", bp3, bp1)
	}

}

func TestMerge(t *testing.T) {
	pnt := bp1
	m := pnt.Merge(bp2)
	if !m.Equals(bp2) {
		t.Errorf("merge(%v, %v) was ¤v.", bp1, bp2, m)
	}
	m = pnt.Merge(bp3)
	if !m.Equals(bp1) {
		t.Errorf("merge(%v, %v) was ¤v.", bp1, bp3, m)
	}
	pnt = bpx
	m = pnt.Merge(bpy)
	if !m.Equals(bpz) {
		t.Errorf("merge(%v, %v) was ¤v.", bpx, bpy, m)
	}
	pnt = nil
	m = pnt.Merge(bpy)
	fmt.Println("nil merge returned ", m)
}

func TestToMsg(t *testing.T) {
	b := GetBlueprint(bp2.ToMsg())
	if !b.Equals(bp2) {
		t.Errorf("Transforming %v to a Protobuf-Msg and back returned %v.", bp2, b)
	}
}

var benchadd = map[ID]bool{ID(1): true, ID(2): true, ID(3): true, ID(4): true, ID(5): true, ID(6): true, ID(7): true}
var benchrem = map[ID]bool{ID(8): true, ID(9): true, ID(10): true, ID(11): true, ID(12): true, ID(13): true, ID(14): true}

var benchbp = Blueprint{benchadd, benchrem}

func BenchmarkToMsg(b *testing.B) {
	GetBlueprint(benchbp.ToMsg())
}
