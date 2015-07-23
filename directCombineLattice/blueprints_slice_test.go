package directCombineLattice

import (
	pb "github.com/relab/smartMerge/proto"
	"testing"
	//"fmt"
)

var onei = uint32(1)
var twoi = uint32(2)
var trei = uint32(3)

var a0i = []uint32{}
var a1i = []uint32{onei, twoi}
var a2i = []uint32{onei, trei}
var r1i = []uint32{onei}
var r2i = []uint32{trei}
var a3i = []uint32{twoi, trei}

func TestSUnion(t *testing.T) {
	u1 := sunion(r1i, r2i)
	for _, id := range u1 {
		if (id != onei) && id != trei {
			t.Error("Union did introduce new element.")
		}
	}
	if len(u1) != 2 {
		t.Error("Union does not have the right number of elements.")
	}

	u1 = sunion(a1i, a2i)

	for _, id := range u1 {
		if id != onei && id != trei && id != twoi {
			t.Error("Union did introduce new element.")
		}
	}
	if len(u1) != 3 {
		t.Error("Union does not have the right number of elements.")
	}
}

func TestSDifference(t *testing.T) {
	df := sdifference(a2i, r2i)
	for _, id := range df {
		if id != onei {
			t.Error("Unwanted element in sdifference.")
		}
	}
	if len(df) != 1 {
		t.Error("Difference does not have the right number of elements.")
	}

	df = sdifference(a1i, a2i)

	if len(df) != 1 {
		t.Error("Difference does not have the right number of elements.")
	}
	for _, id := range df {
		if id != twoi {
			t.Errorf("Difference(%v,%v) was %v, not ID(2).", a1i, a2i, df)
		}
	}
}

func TestSSubset(t *testing.T) {
	if !ssubset(r1i, a1i) {
		t.Errorf("%v was no ssubset of %v.", r1i, a1i)
	}
	if ssubset(a2i, a1i) {
		t.Errorf("%v was ssubset of %v.", a2i, a1i)
	}
}

var bpi1 = &pb.Blueprint{r1i, r2i}
var bpi2 = &pb.Blueprint{a1i, r2i}
var bpi3 = &pb.Blueprint{a2i, a0i}
var bpix = &pb.Blueprint{r2i, r1i}
var bpiy = &pb.Blueprint{a1i, a0i}
var bpiz = &pb.Blueprint{a3i, r1i}

func TestSCompar(t *testing.T) {
	if Compare(bpi1, bpi2) != 1 {
		t.Errorf("%v not smaller %v.", bpi1, bpi2)
	}
	if Compare(bpi2, bpi1) != -1 {
		t.Errorf("%v not larger %v.", bpi2, bpi1)
	}
	if Compare(bpix, bpiy) != 0 {
		t.Errorf("%v comparable to %v.", bpix, bpiy)
	}
	if Compare(bpi3, bpi1) != 1 {
		t.Errorf("%v not larger %v.", bpi3, bpi1)
	}

}

func TestSMerge(t *testing.T) {
	m := Merge(bpi1, bpi2)
	if !Equals(m, bpi2) {
		t.Errorf("merge(%v, %v) was ¤v.", bpi1, bpi2, m)
	}
	m = Merge(bpi1, bpi3)
	if !Equals(m, bpi1) {
		t.Errorf("merge(%v, %v) was ¤v.", bpi1, bpi3, m)
	}
	m = Merge(bpix, bpiy)
	if !Equals(m, bpiz) {
		t.Errorf("merge(%v, %v) was ¤v.", bpi1, bpi2, m)
	}
}

// var r1 = map[ID]bool{ID(1): true}
// var r2 = map[ID]bool{ID(3): true}
// var bp1 = &Blueprint{r1, r2}

func TestToMsg2(t *testing.T) {
	b := GetBlueprint(bpi1)
	if !b.Equals(bp1) {
		t.Errorf("GetBlueprint returned %v, from %v\n", b, bpi1)
	}
	bi := bp1.ToMsg()
	if !Equals(bi, bpi1) {
		t.Errorf("ToMsg returned %v, from %v\n", bi, bp1)
	}
}
