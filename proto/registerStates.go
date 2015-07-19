package proto

func (s *State) Compare(st *State) int {
	if s == nil && st == nil {
		return 0
	}
	if s == nil {
		return 1
	}
	if st == nil {
		return -1
	}
	if s.Timestamp < st.Timestamp {
		return 1
	}
	if s.Timestamp > st.Timestamp {
		return -1
	}

	// Here s.T == st.T holds.
	if s.Writer < st.Writer {
		return 1
	}
	if s.Writer > st.Writer {
		return -1
	}

	return 0
}
