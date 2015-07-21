package directCombineLattice

type ID uint32

type Blueprint struct {
	Add map[ID]bool
	Rem map[ID]bool
}

func union(A, B map[ID]bool) (C map[ID]bool) {
	C = make(map[ID]bool, len(A))
	for id := range B {
		C[id] = true
	}
	for id := range A {
		C[id] = true
	}
	return C
}

func difference(A, B map[ID]bool) (C map[ID]bool) {
	C = make(map[ID]bool, len(A))
	for id := range A {
		if _, ok := B[id]; !ok {
			C[id] = true
		}
	}
	return C
}

func subset(A, B map[ID]bool) bool {
	for id := range A {
		if _, ok := B[id]; !ok {
			return false
		}
	}
	return true
}

func (bp *Blueprint) AddP(id ID) {
	if bp == nil {
		bp = new(Blueprint)
	}
	if bp.Add == nil {
		bp.Add = map[ID]bool{id: true}
		return
	}
	bp.Add[id] = true
}

func (bp *Blueprint) RemP(id ID) {
	if bp == nil {
		bp = new(Blueprint)
	}
	if bp.Rem == nil {
		bp.Rem = make(map[ID]bool,1)
	}
	bp.Rem[id] = true 
	if bp.Add != nil {
		bp.Add = difference(bp.Add, bp.Rem)
	}
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

func (bp *Blueprint) Ids() []uint32 {
	if bp == nil {
		return nil
	}
	ids := make([]uint32,0,len(bp.Add))
	for id, _ := range bp.Add {
		ids = append(ids, uint32(id))
	}
	return ids
}
