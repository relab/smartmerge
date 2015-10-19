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

