package proto

func Union(A, B []uint32) (C []uint32) {
	return union(A, B)
}

func union(A, B []uint32) (C []uint32) {
	C = make([]uint32, len(A))
	copy(C, A)
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

func Intersection(A, B []uint32) (C []uint32) {
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
	return difference(A, B)
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
	mbp.Nodes = make([]*Node, len(bp.Nodes))
	for i,n := range bp.Nodes {
		mbp.Nodes[i] = &Node{Id: n.Id, Version: n.Version}
	}

	for _, n := range blpr.Nodes {
		found := false
	for_blpr:
		for _, node := range mbp.Nodes {
			if n.Id == node.Id {
				found = true
				if n.Version >= node.Version {
					node.Version = n.Version
				} else {
					break for_blpr
				}
			}
		}
		if !found {
			mbp.Nodes = append(mbp.Nodes, &Node{Id: n.Id, Version: n.Version})
		}
	}

	switch {
	case bp.Epoch > blpr.Epoch:
		mbp.Epoch = bp.Epoch
		mbp.FaultTolerance = bp.FaultTolerance
	case blpr.Epoch > blpr.Epoch:
		mbp.Epoch = blpr.Epoch
		mbp.FaultTolerance = blpr.FaultTolerance
	case bp.FaultTolerance > blpr.FaultTolerance:
		mbp.Epoch = bp.Epoch
		mbp.FaultTolerance = bp.FaultTolerance
	default:
		mbp.Epoch = blpr.Epoch
		mbp.FaultTolerance = blpr.FaultTolerance
	}
	return mbp
}

// a.Compare b = 1 <=> a <= b
// a.Compare b = -1 <=> b < a
// a.Compare b = 0 <=> !(b <= a) && !(a <= b)
func (a *Blueprint) Compare(b *Blueprint) int {
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	aleqb := true
	bleqa := true

	switch {
	case a.Epoch > b.Epoch:
		aleqb = false
	case b.Epoch > a.Epoch:
		bleqa = false
	case a.FaultTolerance > b.FaultTolerance:
		aleqb = false
	case b.FaultTolerance > a.FaultTolerance:
		bleqa = false
	}

	if aleqb {
	for_a:
		for _, na := range a.Nodes {
			found := false
		for_b:
			for _, nb := range b.Nodes {
				if na.Id == nb.Id {
					found = true
					if na.Version > nb.Version {
						aleqb = false
						break for_a
					}
					if na.Version < nb.Version {
						bleqa = false
					}
					break for_b
				}
			}
			if !found {
				aleqb = false
				break for_a
			}
		}
	}

	if bleqa {
	for_B:
		for _, nb := range b.Nodes {
			found := false
		for_A:
			for _, na := range a.Nodes {
				if nb.Id == na.Id {
					found = true
					if nb.Version > na.Version {
						bleqa = false
						break for_B
					}
					break for_A
				}
			}
			if !found {
				bleqa = false
				break for_B
			}
		}
	}

	if !aleqb && !bleqa {
		return 0
	}
	if aleqb {
		return 1
	}
	return -1
}

func (bp *Blueprint) Equals(blpr *Blueprint) bool {
	return bp.Compare(blpr) == 1 && blpr.Compare(bp) == 1
}

// See Ids.
func (bp *Blueprint) Len() int {
	if bp == nil {
		return 0
	}

	if bp.FaultTolerance > uint32(15) {
		panic("Specified Fault tolerance larger than 15. Len nor correct for such values.")
	}

	sum := uint32(0)
	for _, n := range bp.Nodes {
		sum += n.Version
	}

	sum += bp.Epoch * 16
	sum += bp.FaultTolerance

	return int(sum)
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

// Oups: Nodes with even version are part of the configuration, those with odd
// 	version have been removed.
func (bp *Blueprint) Ids() []uint32 {
	if bp == nil {
		return nil
	}
	ids := make([]uint32, 0, len(bp.Nodes))
	for _, n := range bp.Nodes {
		if n.Version%2 == 0 {
			ids = append(ids, n.Id)
		}
	}
	return ids
}

// Returns true, if node was added, false, if node was already present.
func (bp *Blueprint) Add(id uint32) bool {
	for _, n := range bp.Nodes {
		if n.Id == id {
			if n.Version%2 == 1 {
				n.Version++
				return true
			}
			// Is already added.
			return false
		}
	}
	bp.Nodes = append(bp.Nodes, &Node{Id: id, Version: uint32(0)})
	return true
}

// Returns true, if node was removed, false otherwise
func (bp *Blueprint) Rem(id uint32) bool {
	for _, n := range bp.Nodes {
		if n.Id == id {
			if n.Version%2 == 0 {
				n.Version++
				return true
			}
			// Is already added.
			return false
		}
	}
	return false
}

func (bp *Blueprint) Quorum() int {
	n := len(bp.Ids())
	if q := n/2 + 1; q >= n-int(bp.FaultTolerance) {
		return q
	}
	return n - int(bp.FaultTolerance)
}

func (bp *Blueprint) Copy() *Blueprint {
	b := new(Blueprint)
	b.Epoch = bp.Epoch
	b.FaultTolerance = bp.FaultTolerance
	b.Nodes = make([]*Node, len(bp.Nodes))
	for i, n := range bp.Nodes {
		b.Nodes[i] = &Node{n.Id, n.Version}
	}
	return b
}
