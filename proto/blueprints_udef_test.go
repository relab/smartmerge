package proto

import (
	"testing"
)

var one = uint32(1)
var two = uint32(2)
var tre = uint32(3)

var a0i []uint32
var a1i = []uint32{one, two}
var a2i = []uint32{one, tre}
var r1i = []uint32{one}
var r2i = []uint32{tre}
var a3i = []uint32{two, tre}

// func TestUnion(t *testing.T) {
// 	u1 := union(r1i, r2i)
// 	for _, id := range u1 {
// 		if (id != one) && id != tre {
// 			t.Error("Union did introduce new element.")
// 		}
// 	}
// 	if len(u1) != 2 {
// 		t.Error("Union does not have the right number of elements.")
// 	}
//
// 	u1 = union(a1i, a2i)
//
// 	for _, id := range u1 {
// 		if id != one && id != tre && id != two {
// 			t.Error("Union did introduce new element.")
// 		}
// 	}
// 	if len(u1) != 3 {
// 		t.Error("Union does not have the right number of elements.")
// 	}
// 	u1 = union(a1i, a0i)
// 	if len(u1) != len(a1i) {
// 		t.Error("Union with nil was not identity.")
// 	}
// 	for i, x := range u1 {
// 		if x != a1i[i] {
// 			t.Error("Union with nil was not identity.")
// 		}
// 	}
// 	u1 = union(a0i, a1i)
// 	if len(u1) != len(a1i) {
// 		t.Error("Union with nil was not identity.")
// 	}
// 	for i, x := range u1 {
// 		if x != a1i[i] {
// 			t.Error("Union with nil was not identity.")
// 		}
// 	}
// }
//
// func TestDifference(t *testing.T) {
// 	df := difference(a2i, r2i)
// 	for _, id := range df {
// 		if id != one {
// 			t.Error("Unwanted element in sdifference.")
// 		}
// 	}
// 	if len(df) != 1 {
// 		t.Error("Difference does not have the right number of elements.")
// 	}
//
// 	df = difference(a1i, a2i)
//
// 	if len(df) != 1 {
// 		t.Error("Difference does not have the right number of elements.")
// 	}
// 	for _, id := range df {
// 		if id != two {
// 			t.Errorf("Difference(%v,%v) was %v, not ID(2).", a1i, a2i, df)
// 		}
// 	}
//
// 	df = difference(a0i, a1i)
//
// 	if len(df) != 0 {
// 		t.Error("Difference from nil got nonempty result.")
// 	}
//
// 	df = difference(a1i, a0i)
//
// 	if len(df) != len(a1i) {
// 		t.Error("Difference with nil was not identity.")
// 	}
// 	for i, x := range df {
// 		if x != a1i[i] {
// 			t.Error("Difference with nil was not identity")
// 		}
// 	}
// }
//
// func TestSubset(t *testing.T) {
// 	if !subset(r1i, a1i) {
// 		t.Errorf("%v was no ssubset of %v.", r1i, a1i)
// 	}
// 	if subset(a2i, a1i) {
// 		t.Errorf("%v was ssubset of %v.", a2i, a1i)
// 	}
// }

var zero = uint32(0)
var four = uint32(4)
var five = uint32(5)
var six = uint32(6)

var n00 = &Node{zero, zero}
var n10 = &Node{one, zero}
var n20 = &Node{two, zero}
var n30 = &Node{tre, zero}
var n40 = &Node{four, zero}
var n50 = &Node{five, zero}
var n60 = &Node{six, zero}

var n11 = &Node{one, one}
var n12 = &Node{one, two}
var n22 = &Node{two, two}
var n32 = &Node{tre, two}
var n33 = &Node{tre, tre}

var b1 = &Blueprint{[]*Node{n11}, one, one}
var b2 = &Blueprint{[]*Node{n22}, two, one}
var b12 = &Blueprint{[]*Node{n11, n22}, two, one}
var b22 = &Blueprint{[]*Node{n11, n22}, two, two}
var b23 = &Blueprint{[]*Node{n11, n22}, tre, two}

var b12x = &Blueprint{[]*Node{n12, n22}, two, one}
var b123 = &Blueprint{[]*Node{n12, n22, n32}, two, one}
var bx = &Blueprint{[]*Node{n11, n33}, tre, two}
var by = &Blueprint{[]*Node{n12, n32}, two, one}
var b0 *Blueprint

var q0 = &Blueprint{[]*Node{n00, n10, n20, n30, n40, n50, n60}, zero, zero}
var q1 = &Blueprint{[]*Node{n00, n10, n20, n30, n40, n50, n60}, one, zero}
var q2 = &Blueprint{[]*Node{n00, n10, n20, n30, n40, n50, n60}, two, zero}
var q3 = &Blueprint{[]*Node{n00, n10, n20, n30, n40, n50, n60}, tre, zero}
var q5 = &Blueprint{[]*Node{n00, n10, n20, n30, n40, n50, n60}, five, zero}

var qx0 = &Blueprint{[]*Node{n00, n11, n20, n30, n40, n50, n60}, zero, zero}
var qx1 = &Blueprint{[]*Node{n00, n11, n20, n30, n40, n50, n60}, one, zero}
var qx2 = &Blueprint{[]*Node{n00, n11, n20, n30, n40, n50, n60}, two, zero}
var qx3 = &Blueprint{[]*Node{n00, n11, n20, n30, n40, n50, n60}, tre, zero}
var qx5 = &Blueprint{[]*Node{n00, n11, n20, n30, n40, n50, n60}, five, zero}

func TestCopy(t *testing.T) {
	cop := b123.Copy()
	if !(cop.Equals(b123)) {
		t.Error("Copy does not equal")
	}
	cop.Rem(two)
	if cop.LearnedEquals(b123) {
		t.Error("Changed copy still equals original")
	}
}

func TestQuorum(t *testing.T) {
	if q0.Quorum() != 7 {
		t.Error("Wrong quorum")
	}
	if q1.Quorum() != 6 {
		t.Error("Wrong quorum")
	}
	if q2.Quorum() != 5 {
		t.Error("Wrong quorum")
	}
	if q3.Quorum() != 4 {
		t.Error("Wrong quorum")
	}
	if q5.Quorum() != 4 {
		t.Error("Wrong quorum")
	}
	if qx0.Quorum() != 6 {
		t.Error("Wrong quorum")
	}
	if qx1.Quorum() != 5 {
		t.Error("Wrong quorum")
	}
	if qx2.Quorum() != 4 {
		t.Error("Wrong quorum")
	}
	if qx3.Quorum() != 4 {
		t.Error("Wrong quorum")
	}
	if qx5.Quorum() != 4 {
		t.Error("Wrong quorum")
	}
}

func TestIds(t *testing.T) {
	if len(b2.Ids()) != 1 {
		t.Error("Unexpected Ids")
	}
	if len(b12.Ids()) != 1 {
		t.Error("Unexpected Ids")
	}
	if len(b12x.Ids()) != 2 {
		t.Error("Unexpected Ids")
	}
	if len(b123.Ids()) != 3 {
		t.Error("Unexpected Ids")
	}
	if len(bx.Ids()) != 0 {
		t.Error("Unexpected Ids")
	}
	if len(b0.Ids()) != 0 {
		t.Error("Unexpected Ids")
	}
	if len(by.Ids()) != 2 {
		t.Error("Unexpected Ids")
	}
}

func TestLearnedCompare(t *testing.T) {
	if b12.LearnedCompare(b12x) != 1 {
		t.Error("Unexpected Comparison")
	}
	if b12x.LearnedCompare(b12) != -1 {
		t.Error("Unexpected Comparison")
	}
	if b1.LearnedCompare(b1) != 0 {
		t.Error("Not equal to self")
	}
	if b1.LearnedCompare(b12) != 1 {
		t.Error("Compare did not find smaller")
	}
	if b12.LearnedCompare(b2) != -1 {
		t.Error("Compare did not find larger")
	}
	if b12.LearnedCompare(b22) != 1 {
		t.Error("Compare did not find smaller")
	}
	if b22.LearnedCompare(b23) != 1 {
		t.Error("Compare did not find smaller")
	}
	if b22.LearnedCompare(b12) != -1 {
		t.Error("Compare did not find larger")
	}
	if b23.LearnedCompare(b22) != -1 {
		t.Error("Compare did not find larger")
	}
	if b0.LearnedCompare(b23) != 1 {
		t.Error("Compare did not find smaller")
	}
	if b22.LearnedCompare(b0) != -1 {
		t.Error("Compare did not find larger")
	}
}

// func TestLen(t *testing.T) {
// 	if b2.Len() != 2+2+16 {
// 		t.Error("Unexpected Length")
// 	}
// 	if b12.Len() != 1+2+2+16 {
// 		t.Error("Unexpected Length")
// 	}
// 	if b12x.Len() != 2+2+2+16 {
// 		t.Error("Unexpected Length")
// 	}
// 	if b123.Len() != 2+2+2+2+16 {
// 		t.Error("Unexpected Length")
// 	}
// 	if bx.Len() != 1+3+3+16*2 {
// 		t.Error("Unexpected Length")
// 	}
// 	if b0.Len() != 0 {
// 		t.Error("Unexpected Length")
// 	}
// 	if by.Len() != 2+2+2+16 {
// 		t.Error("Unexpected Length")
// 	}
//
// }

func TestMerge(t *testing.T) {
	if !b1.Merge(b2).Equals(b12) {
		t.Error("Unexpected Merge")
	}
	if !b2.Merge(b12).Equals(b12) {
		t.Error("Unexpected Merge")
	}
	if !b1.Merge(b22).Equals(b22) {
		t.Error("Unexpected Merge")
	}
	if !b12.Merge(by).Equals(b123) {
		t.Error("Unexpected Merge")
	}
	if !bx.Merge(b0).Equals(bx) {
		t.Error("Unexpected Merge")
	}
	if !b0.Merge(bx).Equals(bx) {
		t.Error("Unexpected Merge")
	}
}

func TestCompare(t *testing.T) {
	if b12.Compare(b12x) != 1 {
		t.Error("Unexpected Comparison")
	}
	if b12x.Compare(b12) != -1 {
		t.Error("Unexpected Comparison")
	}
	if b12x.Compare(b22) != 0 {
		t.Error("Unexpected Comparison")
	}
	if b1.Compare(b1) != 1 {
		t.Error("Not equal to self")
	}
	if b1.Compare(b2) != 0 {
		t.Error("Compare did not find uncomparable")
	}
	if b1.Compare(b12) != 1 {
		t.Error("Compare did not find smaller")
	}
	if b12.Compare(b2) != -1 {
		t.Error("Compare did not find larger")
	}
	if b12.Compare(b22) != 1 {
		t.Error("Compare did not find smaller")
	}
	if b22.Compare(b23) != 1 {
		t.Error("Compare did not find smaller")
	}
	if b22.Compare(b12) != -1 {
		t.Error("Compare did not find larger")
	}
	if b23.Compare(b22) != -1 {
		t.Error("Compare did not find larger")
	}
	if b0.Compare(b23) != 1 {
		t.Error("Compare did not find smaller")
	}
	if b22.Compare(b0) != -1 {
		t.Error("Compare did not find larger")
	}
}

func TestEquals(t *testing.T) {
	if !b1.Equals(b1) {
		t.Error("Not equal to itself")
	}
	if !b2.Equals(b2) {
		t.Error("Not equal to itself")
	}
	if !b12.Equals(b12) {
		t.Error("Not equal to itself")
	}
	if !b0.Equals(b0) {
		t.Error("Not equal to itself")
	}
	if b1.Equals(b2) {
		t.Error("Not similar, but equal")
	}
	if b2.Equals(b12) {
		t.Error("Not similar, but equal")
	}
	if b12.Equals(b22) {
		t.Error("Not similar, but equal")
	}
	if b22.Equals(b23) {
		t.Error("Not similar, but equal")
	}
	if b2.Equals(b1) {
		t.Error("Not similar, but equal")
	}
	if b12.Equals(b2) {
		t.Error("Not similar, but equal")
	}
	if b22.Equals(b12) {
		t.Error("Not similar, but equal")
	}
	if b23.Equals(b22) {
		t.Error("Not similar, but equal")
	}
	if b1.Equals(b0) {
		t.Error("Not similar, but equal")
	}
	if b2.Equals(b0) {
		t.Error("Not similar, but equal")
	}
	if b0.Equals(b22) {
		t.Error("Not similar, but equal")
	}
	if b22.Equals(b0) {
		t.Error("Not similar, but equal")
	}
}

// Oups: This test has side effects.
func TestAddRem(t *testing.T) {
	if !b2.Add(one) {
		t.Error("Add did not signal true")
	}
	if !b2.Rem(one) {
		t.Error("Add did not signal true")
	}
	if !b2.Equals(b12) {
		t.Error("Wrong result from adding & removing")
	}
	b12.Add(one)
	if !b12.Equals(b12x) {
		t.Error("Wrong result from adding & removing")
	}
}
