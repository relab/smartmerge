package proto

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

func (m *Manager) GetErrors() map[uint32]error {
	err := make(map[uint32]error, len(m.machines))
	for _, ma := range m.machines {
		if ma.lastErr != nil {
			err[ma.gid] = ma.lastErr
		}
	}
	return err
}

func (m *Manager) ToIds(gids []uint32) (ids []int) {
	ids = make([]int,len(gids))
	for i, gid := range gids {
		ids[i] = m.machineGidToID[gid]
	}
	return ids
}
