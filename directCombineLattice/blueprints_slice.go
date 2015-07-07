package directCombineLattice

import (
	pb "github.com/relab/smartMerge/proto"
)

func sunion(A, B []uint32) (C []uint32) {
	C = make([]uint32, 0, len(A))
	for _, id := range A {
		C = append(C, id)
	}
	for _, id := range B {
		copy := true
		for _, id2 := range A {
			if id == id2 {
				copy = false
				break
			}
		}
		if copy {
			C = append(C, id)
		}
	}
	return C
}

func sdifference(A, B []uint32) (C []uint32) {
	C = make([]uint32, 0, len(A))
	for _, id := range A {
		copy := true
		for _, id2 := range B {
			if id == id2 {
				copy = false
				break
			}
		}
		if copy {
			C = append(C, id)
		}
	}
	return C
}

func ssubset(A, B []uint32) bool {
	for _, id := range A {
		exists := false
		for _, id2 := range B {
			if id == id2 {
				exists = true
				break
			}
		}
		if !exists {
			return false
		}
	}
	return true
}

func Merge(bp, blpr pb.Blueprint) (mbp pb.Blueprint) {
	mbp.Rem = sunion(bp.Rem, blpr.Rem)
	mbp.Add = sdifference(sunion(bp.Add, blpr.Add), mbp.Rem)
	return mbp
}

// a.Compare b = 1 <=> a <= b
// a.Compare b = -1 <=> b < a
// a.Compare b = 0 <=> !(b <= a) && !(a <= b)
func Compare(bp, blpr pb.Blueprint) int {
	if ssubset(bp.Add, sunion(blpr.Add, blpr.Rem)) && ssubset(bp.Rem, blpr.Rem) {
		return 1
	}
	if ssubset(blpr.Add, sunion(bp.Rem, bp.Add)) && ssubset(blpr.Rem, bp.Rem) {
		return -1
	}
	return 0
}

func Equals(bp, blpr pb.Blueprint) bool {
	return Compare(bp, blpr) == 1 && Compare(blpr, bp) == 1
}
