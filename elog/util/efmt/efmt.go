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

func main() {
	var file = flag.String("file", "", "elog files to parse, separated by comma")
	//var filter = flag.Bool("filter", true, "filter out throughput samples")
	var outfile = flag.String("outfile", "", "write results to file")
	var list = flag.Bool("list", false, "print a list or latencies")
	var debug = flag.Bool("debug", false, "print spike latencies")
	var norm = flag.Int("normal", 2, "number of accesses in normal case.")
	var recs = flag.Int("recs", 1, "number of reconfigurations per run.")
	var cl = flag.Int("clients", 5, "number of clients.")

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
	events := make([]e.Event, 0, len(infiles))
	eventsperc := make([][]e.Event, len(infiles))

	for i, fi := range infiles {
		if fi == "" {
			continue
		}
		fievents, err := e.Parse(fi)
		if err != nil {
			fmt.Printf("Error %v  parsing events from %v", err, fi)
			return
		}
		events = append(events, fievents...)
		eventsperc[i] = fievents
	}

	if *debug {
		//fmt.Fprintf(of, "%v\n", events[0])
		cnt := 0
		spikes := make([]e.Event, 0)
		recs := make([]e.Event, 0, 10)
		i := 0
		for _, evt := range events {
			if evt.EndTime.Sub(evt.Time) > 100*time.Millisecond {
				fmt.Fprintf(of, "%v\n", evt)
				cnt++
				spikes = append(spikes, evt)
			}
			if evt.Type == e.ClientReconfLatency && i < 20 {
				fmt.Fprintf(of, "%v\n", evt)
				recs = append(recs, evt)
				i++
			}
		}

		//fmt.Fprintf(of, "%v\n", events[len(events)-1])
		fmt.Fprintf(of, "%d spike latencies.\n", cnt)

		if len(recs) > 0 && len(spikes) > 0 {
			start := recs[0].Time
			end := recs[0].EndTime
			for _, rec := range recs {
				if rec.Time.Before(start) {
					start = rec.Time
				}
				if rec.EndTime.After(end) {
					end = rec.EndTime
				}
			}
			end = end.Add(100 * time.Millisecond)

			problem := false
			for _, evt := range spikes {
				if evt.Value > uint64(*norm) {
					problem = true
					break
				}
				if evt.EndTime.Add(100*time.Millisecond).After(start) && evt.Time.Before(end) {
					fmt.Fprintln(of, "Spike during reconfiguration:")
					fmt.Fprintf(of, "%v end time: %v\n", evt, evt.EndTime)
					problem = true
					break
				}
			}
			if problem {
				os.Exit(1)
			}
		}

		return
	}

	if *list {
		fmt.Printf("Type of first event: %v", events[0].Type)
		for _, evt := range events {
			_, err := fmt.Fprintf(of, "%d: %d\n", evt.Value, evt.EndTime.Sub(evt.Time))
			if err != nil {
				fmt.Println("Error writing to file:", err)
			}
		}
		return
	}

	reade, writee, reconfe, tupute := sortLatencies(events)

	normal := uint64(*norm)
	if len(reade) > 0 {
		var totalaffected time.Duration
		var affected time.Duration

		_, err := fmt.Fprintln(of, "Read Latencies:")
		if err != nil {
			fmt.Println("Error writing to file:", err)
		}
		readl := makeMap(reade)
		avgWrites := computeAverageDurations(readl)
		for k, durs := range readl {
			fmt.Fprintf(of, "Accesses %2d, %5d times, AvgLatency: %v\n", k, len(durs), avgWrites[k])
			if k != normal {
				affected += time.Duration(len(durs))
				totalaffected += time.Duration(len(durs)) * avgWrites[k]
			}
		}
		if affected > 0 {
			fmt.Fprintf(of, "Mean latency for reads with more than %d acceses is: %v\n", normal, (totalaffected / affected))
			nwavg := 3806 * time.Microsecond
			if normal == 4 {
				nwavg = 5700 * time.Microsecond
			}
			if normal == 1 {
				nwavg = 1870 * time.Microsecond
			}
			overhead := totalaffected - (affected * nwavg)
			fmt.Fprintf(of, "Total overhead is: %v\n", overhead)
			runs := 0
			runs += len(reconfe)
			runs = runs / *recs
			fmt.Fprintf(of, "Number of runs is: %d\n", runs)
			if runs > 0 {
				fmt.Fprintf(of, "Overhead per client is: %v\n", (overhead/time.Duration(*cl))/time.Duration(runs))
			}
		}
		maxls := make([]time.Duration, 0, len(eventsperc))
		for _, es := range eventsperc {
			var max time.Duration
			for _, evt := range es {
				if evt.Type != e.ClientReadLatency {
					break
				}
				if evt.Value <= normal {
					continue
				}
				if evt.EndTime.Sub(evt.Time) > max {
					max = evt.EndTime.Sub(evt.Time)
				}
			}
			maxls = append(maxls, max)
		}
		if len(maxls) > 0 {
			fmt.Fprintf(of, "Mean of maximum read latency: %v\n", MeanDuration(maxls...))
		}

	}

	if len(writee) > 0 {
		fmt.Println("Processing writes is outdated!")
		/*
			var totalaffected time.Duration
			var affected time.Duration

			_, err := fmt.Fprintln(of, "Write Latencies:")
			if err != nil {
				fmt.Println("Error writing to file:", err)
			}
			avgWrites := computeAverageDurations(writel)
			for k, durs := range writel {
				fmt.Fprintf(of, "Accesses %2d, %5d times, AvgLatency: %v\n", k, len(durs), avgWrites[k])
				if k != normal {
					affected += time.Duration(len(durs))
					totalaffected += time.Duration(len(durs)) * avgWrites[k]
				}
			}
			if affected > 0 {
				fmt.Fprintf(of, "Mean latency for writes with more than %d acceses is: %v\n", normal, (totalaffected / affected))
				nwavg := 3300 * time.Microsecond
				if normal == 4 {
					nwavg = 5700 * time.Microsecond
				}

				fmt.Fprintf(of, "Total overhead is: %v\n", totalaffected-(affected*nwavg))
			}*/
	}

	if len(reconfe) > 0 {
		reconfl := makeMap(reconfe)
		var total time.Duration
		var number time.Duration
		_, err := fmt.Fprintln(of, "Reconf Latencies:")
		if err != nil {
			fmt.Println("Error writing to file:", err)
		}
		for k, durs := range reconfl {
			avg := MeanDuration(durs...)
			fmt.Fprintf(of, "Accesses %2d, %5d times, AvgLatency: %v\n", k, len(durs), avg)
			total += avg * time.Duration(len(durs))
			number += time.Duration(len(durs))
		}
		fmt.Fprintf(of, "Average reconfiguration latency: %v\n", (total / number))
		fmt.Fprintf(of, "In total: %d reconfigurations\n", number)
	}

	if len(tupute) > 0 {

		tupute = combineTPut(tupute)
		fmt.Fprintf(of, "Throuputs: %d measurepoints.", len(tupute))
		PrintTputsAndReconfs(tupute, reconfe, of)
	}

}

func sortLatencies(events []e.Event) (reade, writee, reconfe, tupute []e.Event) {
	reade = make([]e.Event, 0, 100)
	writee = make([]e.Event, 0, 100)
	reconfe = make([]e.Event, 0, 100)
	tupute = make([]e.Event, 0, 100)

	for _, evt := range events {
		if evt.EndTime.Sub(evt.Time) > 100*time.Millisecond {
			fmt.Printf("Discarding event %v.\n", evt)
			continue
		}

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

func makeMap(events []e.Event) (dmap map[uint64][]time.Duration) {
	dmap = make(map[uint64][]time.Duration, 0)
	for _, evt := range events {
		if dmap[evt.Value] == nil {
			dmap[evt.Value] = []time.Duration{evt.EndTime.Sub(evt.Time)}
		} else {
			dmap[evt.Value] = append(dmap[evt.Value], evt.EndTime.Sub(evt.Time))
		}
	}
	return
}

func computeAverageDurations(durs map[uint64][]time.Duration) map[uint64]time.Duration {
	if durs == nil {
		return nil
	}
	avgs := make(map[uint64]time.Duration, len(durs))
	for k, ds := range durs {
		avgs[k] = MeanDuration(ds...)
	}
	return avgs
}

type durarr []time.Duration

func (a durarr) Len() int           { return len(a) }
func (a durarr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a durarr) Less(i, j int) bool { return a[i] < a[j] }

func MedianDuration(v ...time.Duration) time.Duration {
	if len(v) == 0 {
		return 0
	}
	da := durarr(v)
	sort.Sort(da)
	return v[len(v)/2]
}

func ComputeMedianNotNormal(durs map[uint64][]time.Duration, normal uint64) time.Duration {
	if durs == nil {
		return time.Duration(0)
	}
	allnotNormal := make([]time.Duration, 0, 100)
	for k, ds := range durs {
		if k != normal {
			for _, dur := range ds {
				allnotNormal = append(allnotNormal, dur)
			}
		}
	}
	return MedianDuration(allnotNormal...)
}

func MeanDuration(v ...time.Duration) time.Duration {
	if len(v) == 0 {
		return 0
	}
	var sum time.Duration
	for _, dur := range v {
		sum += dur
	}
	return sum / time.Duration((len(v)))
}

type evtarr []e.Event

func (a evtarr) Len() int           { return len(a) }
func (a evtarr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a evtarr) Less(i, j int) bool { return a[i].Time.Before(a[j].Time) }

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

func PrintTputsAndReconfs(tpute, reconfe []e.Event, of io.Writer) {
	rar := evtarr(reconfe)
	sort.Sort(rar)

	i := 0
	for _, tput := range tpute {
		count := 0
	for_rec:
		for i < len(reconfe) {
			if reconfe[i].Time.Before(tput.Time) {
				count++
				i++
			} else {
				break for_rec
			}
		}
		fmt.Fprintf(of, "Initialized %d reconfigurations before: ", count)
		fmt.Fprintf(of, "%v\n", tput)
	}
}
