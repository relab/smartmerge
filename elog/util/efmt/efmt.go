package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	e "github.com/relab/smartMerge/elog/event"
)

var normlat time.Duration

func main() {
	var file = flag.String("file", "", "elog files to parse, separated by comma")
	var outfile = flag.String("outfile", "", "write results to file")
	var norm = flag.Int("normal", 2, "number of accesses in normal case.")
	var normL = flag.Int("normlat", -1, "normal case latency in milliseconds.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *file == "" {
		flag.Usage()
		os.Exit(1)
	}

	var of io.Writer
	if *outfile == "" {
		of = os.Stderr
	} else {
		fl, err := os.OpenFile(*outfile, os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println("Could not open file, create.")
			fl, err = os.Create(*outfile)
			if err != nil {
				fmt.Println("Could not create file: ", *outfile)
				return
			}
		}
		defer fl.Close()
		of = fl
	}

	infiles := strings.Split(*file, ",")

	resultChan := make(chan *EvaluationResult, len(infiles))
	for _, fi := range infiles {
		if fi == "" {
			continue
		}
		go func(fi string, norm, normL int) {
			fievents, err := e.Parse(fi)
			if err != nil {
				fmt.Printf("Error %v  parsing events from %v", err, fi)
				resultChan <- nil
			}
			r := computeResult(fievents, norm, normL)
			resultChan <- &r
		}(fi, *norm, *normL)
	}

	results := make([]*EvaluationResult, 0, len(infiles))
	var r *EvaluationResult
	for i := 0; i < len(infiles); i++ {
		r = <-resultChan
		if r != nil {
			results = append(results, r)
		}

		combine(results)
		fmt.Fprint(of, results)

	}
}

func combine(eresults []*EvaluationResult) *EvaluationResult {
	r := eresults[0]
	for i := 1; i < len(eresults); i++ {
		r.reads.combine(&eresults[i].reads)
		r.writes.combine(&eresults[i].writes)
		r.reconfs.combine(&eresults[i].reconfs)
		r.tput = combineTPut(append(r.tput, eresults[i].tput...))
	}
	return r

}

func (rt Roundtrip) combine(nrt Roundtrip) Roundtrip {
	if rt.n == 0 {
		rt.n = nrt.n
	} else if rt.n != nrt.n {
		return rt
	}
	rt.avgLat = (rt.avgLat * time.Duration(rt.count)) + (nrt.avgLat*time.Duration(nrt.count))/time.Duration(rt.count+nrt.count)
	rt.count = rt.count + nrt.count
	return rt
}

func (r *Result) combine(nr *Result) {
	if r.normalRoundtrips != nr.normalRoundtrips {
		fmt.Print("Trying to combine result with different normal round trips.")
		return
	}
	for _, nrt := range nr.perRoundtripData {
		r.perRoundtripData[nrt.n] = r.perRoundtripData[nrt.n].combine(nrt)
	}
	r.avgNotNormal = r.perRoundtripData[-1].avgLat
	r.overhead = append(r.overhead, nr.overhead...)
	r.maxLatency = append(r.maxLatency, nr.maxLatency...)
}

func (r *RecResult) combine(nr *RecResult) {
	for _, nrt := range nr.perRoundtripData {
		r.perRoundtripData[nrt.n] = r.perRoundtripData[nrt.n].combine(nrt)
	}
	r.totalAvgLatency = r.perRoundtripData[-1].avgLat

}

type evtarr []e.Event

func (a evtarr) Len() int           { return len(a) }
func (a evtarr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a evtarr) Less(i, j int) bool { return a[i].Time.Before(a[j].Time) }

//Takes an array of throughput samples and combines them.
//The output will include at most one throughput sample per 100 ms.
func combineTPut(events []e.Event) (tputs []e.Event) {
	if len(events) == 0 {
		return nil
	}

	evts := evtarr(events)
	sort.Sort(evts)
	tputs = []e.Event{events[0]}
	for i := 1; i < len(events); i++ {
		if events[i].Type != e.ThroughputSample {
			return nil
		}

		if events[i].Time.Sub(tputs[len(tputs)-1].Time) < 100*time.Millisecond {
			tputs[len(tputs)-1].Value += events[i].Value
		} else {
			tputs = append(tputs, events[i])
		}
	}
	return tputs
}

func mean64(v []uint64) uint64 {
	if len(v) == 0 {
		return 0
	}
	var sum uint64
	for _, x := range v {
		sum += x
	}
	return sum / uint64(len(v))
}
