package proto

func Union(A, B []uint32) (C []uint32) {
	return union(A,B)
}

func union(A, B []uint32) (C []uint32) {
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

func Intersection(A,B []uint32) (C []uint32) {
	C = make([]uint32, 0, len(A))
	for _, id := range A {
		for _, id2 := range B {
			if id == id2 {
				C = append(C, id)
				break
			}
		}
	}
	return C
}

func Difference(A, B []uint32) (C []uint32) {
	return difference(A,B)
}
func difference(A, B []uint32) (C []uint32) {
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

func subset(A, B []uint32) bool {
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

func (bp *Blueprint) Merge(blpr *Blueprint) (mbp *Blueprint) {
	if bp == nil {
		return blpr
	}
	if blpr == nil {
		return bp
	}
	mbp = new(Blueprint)
	mbp.Rem = union(bp.Rem, blpr.Rem)
	mbp.Add = difference(union(bp.Add, blpr.Add), mbp.Rem)
	return mbp
}

// a.Compare b = 1 <=> a <= b
// a.Compare b = -1 <=> b < a
// a.Compare b = 0 <=> !(b <= a) && !(a <= b)
func (bp *Blueprint) Compare(blpr *Blueprint) int {
	if bp == nil {
		return 1
	}
	if blpr == nil {
		return -1
	}
	if subset(bp.Add, union(blpr.Add, blpr.Rem)) && subset(bp.Rem, blpr.Rem) {
		return 1
	}
	if subset(blpr.Add, union(bp.Rem, bp.Add)) && subset(blpr.Rem, bp.Rem) {
		return -1
	}
	return 0
}

func (bp *Blueprint) Equals(blpr *Blueprint) bool {
	return bp.Compare(blpr) == 1 && blpr.Compare(bp) == 1
}

func (bp *Blueprint) Len() int {
	if bp == nil {
		return 0
	}
	
	return len(bp.Add) + (2 * len(bp.Rem))
}

func (bp *Blueprint) LearnedCompare(blpr *Blueprint) int {
	if bp.Len() < blpr.Len() {
		return 1
	}
	if bp.Len() > blpr.Len() {
		return -1
	}
	
	return 0
}

func (bp *Blueprint) LearnedEquals(blpr *Blueprint) bool {
	return bp.Len() == blpr.Len()
}

func (bp *Blueprint) Ids() []uint32 {
	ids := make([]uint32,0,len(bp.Add))
	for i, id := range bp.Add {
		ids[i]=id
	}
	return ids
}