package main

import (
	"fmt"
	"sort"
	"time"

	e "github.com/relab/smartmerge/elog/event"
)

// Roundtrip repots the avg latency and number of request that performed n round trips.
type Roundtrip struct {
	n      int
	count  int
	avgLat time.Duration
}

// Result for one operation type, read write or reconf.
type Result struct {
	perRoundtripData map[int]Roundtrip
	normalRoundtrips int
	avgNotNormal     time.Duration
	maxLatency       []time.Duration
	overhead         []time.Duration
}

type RecResult struct {
	perRoundtripData map[int]Roundtrip
	totalAvgLatency  time.Duration
}

type EvaluationResult struct {
	reads   Result
	writes  Result
	reconfs RecResult
	tput    []e.Event
}

func computeResult(latencies []e.Event, normal int, normalLat int) EvaluationResult {
	r := EvaluationResult{}
	reade, writee, reconfe, tupute := sortLatencies(latencies)
	r.reads = processReadsWrites(reade, normal, normalLat)
	r.writes = processReadsWrites(writee, normal, normalLat)
	r.reconfs = processRecs(reconfe)
	r.tput = tupute
	return r
}

func processRecs(latencies []e.Event) RecResult {
	if latencies == nil || len(latencies) == 0 {
		return RecResult{}
	}
	perRoundTripLats := makeMap(latencies, -1)
	r := RecResult{}
	r.perRoundtripData = make(map[int]Roundtrip, len(perRoundTripLats)-1)
	for i, lats := range perRoundTripLats {
		r.perRoundtripData[int(i)] = Roundtrip{
			n:      int(i),
			count:  len(lats),
			avgLat: meanDuration(lats...),
		}
	}
	r.totalAvgLatency = r.perRoundtripData[-1].avgLat

	return r
}

func processReadsWrites(latencies []e.Event, normal int, normalLat int) Result {
	if latencies == nil || len(latencies) == 0 {
		return Result{normalRoundtrips: -2}
	}
	perRoundTripLats := makeMap(latencies, normal)
	r := Result{normalRoundtrips: normal}
	r.perRoundtripData = make(map[int]Roundtrip, len(perRoundTripLats)-1)
	for i, lats := range perRoundTripLats {
		r.perRoundtripData[int(i)] = Roundtrip{
			n:      int(i),
			count:  len(lats),
			avgLat: meanDuration(lats...),
		}
	}
	if normal != 1000 {
		notNormal := perRoundTripLats[-1]
		r.avgNotNormal = r.perRoundtripData[-1].avgLat
		nl := r.perRoundtripData[normal].avgLat
		if normalLat != -1 {
			nl = time.Duration(normalLat) * time.Millisecond
		}
		notnormalCount := time.Duration(r.perRoundtripData[-1].count)
		r.overhead = []time.Duration{r.avgNotNormal*notnormalCount - nl*notnormalCount}
		max := 0 * time.Millisecond
		for _, d := range notNormal {
			if d > max {
				max = d
			}
		}
		r.maxLatency = []time.Duration{max}
	}
	return r
}

func meanDuration(v ...time.Duration) time.Duration {
	if len(v) == 0 {
		return 0
	}
	var sum time.Duration
	for _, dur := range v {
		sum += dur
	}
	return sum / time.Duration((len(v)))
}

func makeMap(events []e.Event, normal int) (dmap map[int][]time.Duration) {
	dmap = make(map[int][]time.Duration, 0)
	dmap[-1] = make([]time.Duration, 0, len(events))
	for _, evt := range events {
		if dmap[int(evt.Value)] == nil {
			dmap[int(evt.Value)] = []time.Duration{evt.EndTime.Sub(evt.Time)}
		} else {
			dmap[int(evt.Value)] = append(dmap[int(evt.Value)], evt.EndTime.Sub(evt.Time))
		}
		if int(evt.Value) > normal {
			dmap[-1] = append(dmap[-1], evt.EndTime.Sub(evt.Time))
		}
	}
	return
}

func sortLatencies(events []e.Event) (reade, writee, reconfe, tupute []e.Event) {
	reade = make([]e.Event, 0, 100)
	writee = make([]e.Event, 0, 100)
	reconfe = make([]e.Event, 0, 100)
	tupute = make([]e.Event, 0, 100)

	for _, evt := range events {

		switch evt.Type {
		case e.ClientReadLatency:
			reade = append(reade, evt)
		case e.ClientWriteLatency:
			writee = append(writee, evt)
		case e.ClientReconfLatency:
			reconfe = append(reconfe, evt)
		case e.ThroughputSample:
			tupute = append(tupute, evt)
		}
	}
	return
}

func (er EvaluationResult) String() string {
	var str string
	if er.reads.normalRoundtrips > -2 {
		str += "***********  Reads results  ***************\n"
		str += er.reads.String()
	}
	if er.writes.normalRoundtrips > -2 {
		str += "***********  Writes results  ***************\n"
		str += er.writes.String()
	}
	if er.reconfs.perRoundtripData != nil && len(er.reconfs.perRoundtripData) > 0 {
		str += "***********  Reconfigurations results  ***************\n"
		str += er.reconfs.String()
	}

	if len(er.tput) > 2 {
		str += "***********  Throughput results  ***************\n"
		str += PrintTputs(er.tput)
	}
	return str
}

func (r Result) String() string {
	var str string
	if len(r.perRoundtripData) > 0 {
		str += "Latencies for different round trips numbers:\n"
	}

	for _, rt := range r.perRoundtripData {
		if rt.n != -1 {
			str += fmt.Sprintf(
				"  %d roundtrips: %d times with average latency %v\n",
				rt.n, rt.count, rt.avgLat)
		}
	}
	str += fmt.Sprintf("\nNormal roundtrip number is %d\n", r.normalRoundtrips)
	str += fmt.Sprintf(
		"  Average latency for not normal: %v (requests with more than normal roundtrips)\n", r.avgNotNormal)
	if len(r.maxLatency) == 1 {
		str += fmt.Sprintf("\nMaximum latency is %v \n", r.maxLatency[0])
	} else {
		str += fmt.Sprintf("\nAverage maximum latency is %v (average over the largest latency from different clients)\n", meanDuration(r.maxLatency...))
		maxdur := durarr(r.maxLatency)
		sort.Sort(maxdur)
		if len(r.maxLatency) > 20 {
			str += fmt.Sprintf("  95%% maximum latency is %v\n", maxdur[(len(maxdur)-1)*19/20])
		} else {
			str += fmt.Sprintf("  highest maximum latency is %v\n", maxdur[(len(maxdur)-1)])
		}
	}
	if len(r.overhead) == 1 {
		str += fmt.Sprintf("\nOverhead is %v \n", r.overhead[0])
	} else {
		str += fmt.Sprintf("\nAverage overhead is %v (average over different clients)\n", meanDuration(r.overhead...))
		ohdur := durarr(r.overhead)
		sort.Sort(ohdur)
		if len(r.overhead) > 20 {
			str += fmt.Sprintf("  95%% overhead is %v\n", ohdur[(len(ohdur)-1)*19/20])
		} else {
			str += fmt.Sprintf("  highest overhead is %v\n", ohdur[(len(ohdur)-1)])
		}
	}

	return str
}

func (r RecResult) String() string {
	var str string
	if len(r.perRoundtripData) > 0 {
		str += "Latencies for different round trips numbers:"
	}

	for _, rt := range r.perRoundtripData {
		str += fmt.Sprintf(
			"  %d roundtrips: %d times with average latency %v\n",
			rt.n, rt.count, rt.avgLat)
	}
	str += fmt.Sprintf(
		"  Average latency for reconfigurations is %v \n", r.totalAvgLatency)

	return str
}

type durarr []time.Duration

func (a durarr) Len() int           { return len(a) }
func (a durarr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a durarr) Less(i, j int) bool { return a[i] < a[j] }

func PrintTputs(tpute []e.Event) string {
	var str string

	tp := evtarr(tpute)
	sort.Sort(tp)

	readTP := make([]uint64, 0, 100)

	for k, tput := range tpute {
		if k < 1 || tput.Time.Sub(tpute[k-1].Time) < 800*time.Millisecond || tput.Time.Sub(tpute[k-1].Time) > 1200*time.Millisecond {
			continue
		}
		readTP = append(readTP, tput.Value)

		str += fmt.Sprintf("%v\n", tput)

	}
	str += fmt.Sprintf("\nMean Throughput is %d per second\n", mean64(readTP))
	return str
}
