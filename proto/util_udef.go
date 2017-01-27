package proto

//TODO: These should be methods on the quorumSpec
func (c *Configuration) ReadQuorum() int {
	return c.Size() - c.Quorum() + 1
}

func (c *Configuration) WriteQuorum() int {
	return c.Quorum()
}

func (c *Configuration) MaxQuorum() int {
	if c.Quorum() > c.ReadQuorum() {
		return c.Quorum()
	}
	return c.ReadQuorum()
}

// Functions below are outdated since gorums now uses only global ids.
/*func (m *Manager) ToIds(gids []uint32) (ids []int) {
	ids = make([]int, len(gids))
	for i, gid := range gids {
		ids[i] = m.machineGidToID[gid]
	}
	return ids
}

func (m *Manager) ToGids(ids []int) (gids []uint32) {
	gids = make([]uint32, len(ids))
	allgids := m.MachineGlobalIDs()
	for k, id := range ids {
		gids[k] = allgids[id]
	}
	return gids
}
*/
