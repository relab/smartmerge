package directCombineLattice

type ID uint32

type ProcSet map[ID]bool

type Blueprint struct {
	Add ProcSet
	Rem ProcSet
}

func (A ProcSet) union(B ProcSet) ProcSet {
	for id := range B {
		A[id] = true
	}
	return A
}
/*
func (A map[ID]bool) difference(B map[ID]bool) map[ID]bool {
	for id := range B {
		delete(A, id)
	}
	return A
}

func (A *map[ID]bool) in(B *map[ID]bool) bool {
	for id := range A {
		if ok, _ := B[id]; !ok {
			return false
		}
	}
	return true
}

func (bp *Blueprint) Merge(blpr Blueprint) (mbp Blueprint) {
	mbp.Rem = bp.Rem.union(blpr.Rem)
	mbp.Add = bp.Add.union(blpr.Add).difference(mbp.Rem)
	return mbp
}

// a.Compare b = 1 <=> a <= b
// a.Compare b = -1 <=> b < a
// a.Compare b = 0 <=> !(b <= a) && !(a <= b)
func (bp *Blueprint) Compare(blpr *Blueprint) int {
	if bp.Add.in(blpr.Add) && pb.Rem.in(blpr.Rem) {
		return 1
	}
	if blpr.Add.in(bp.Add) && plbr.Rem.in(bp.Rem) {
		return -1
	}
	return 0
}
*/