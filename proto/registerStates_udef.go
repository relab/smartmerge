package proto

// To gernerate the code from gorums run go generate in this folder
//go:generate protoc -I=../../../../:. --gorums_out=plugins=grpc+gorums:. dc-smartMerge.proto

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
