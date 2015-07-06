package rpc

import (
	"sort"
	"time"
)

type LatencyTuple struct {
	machineID  uint32
	reqLatency time.Duration
}

type latencySlice []LatencyTuple

func (p latencySlice) Len() int           { return len(p) }
func (p latencySlice) Less(i, j int) bool { return p[i].reqLatency < (p[j].reqLatency) }
func (p latencySlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p latencySlice) Sort() { sort.Sort(p) }

func sortLatencies(lats map[uint32]LatencyTuple) latencySlice {
	ls := make(latencySlice, len(lats))
	i := 0
	for _, lat := range lats {
		ls[i] = lat
		i++
	}
	ls.Sort()
	return ls
}
