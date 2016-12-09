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
	//var filter = flag.Bool("filter", true, "filter out throughput samples")
	var outfile = flag.String("outfile", "", "write results to file")
	var list = flag.Bool("list", false, "print a list or latencies")
	var debug = flag.Bool("debug", false, "print spike latencies")
	var norm = flag.Int("normal", 2, "number of accesses in normal case.")
	var normL = flag.Int("normlat", 0, "normal case latency.")
	//var recs = flag.Int("recs", 1, "number of reconfigurations per run.")
	//var cl = flag.Int("clients", 5, "number of clients.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	normlat = time.Duration(*normL) * time.Microsecond

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
			if evt.EndTime.Sub(evt.Time) > 5000*time.Millisecond {
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

			if k != normal && k != 1000 {
				affected += time.Duration(len(durs))
				totalaffected += time.Duration(len(durs)) * avgWrites[k]
			} else if normlat == 0 {
				normlat = avgWrites[k]
			}
		}
		durs := readl[1]
		sort.Sort(durarr(durs))
		fmt.Fprintf(of, "Len: %d Perc %d\n", len(durs), len(durs)*19/20)
		if len(durs) > 5 {
			fmt.Fprintf(of, "For %d accesses, 95perc is %v\n", 2, durs[(len(durs)*19)/20])
		} else if len(durs) > 0 {
			fmt.Fprintf(of, "For %d accesses, max duration is %v\n", 2, durs[len(durs)-1])
		}

		allds := readl[1000]
		sort.Sort(durarr(allds))
		fmt.Fprintf(of, "All read 95perc is %v\n", allds[(len(allds)*19)/20])

		if affected > 0 {
			fmt.Fprintf(of, "Mean latency for reads with more than %d acceses is: %v\n", normal, (totalaffected / affected))
		}
	}

	if len(writee) > 0 {
		fmt.Println("Processing writes is outdated!")
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

		recds := reconfl[1000]
		/*p := 0
		//fmt.Println("Len reconfe is", len(reconfe))
		for k, durs := range reconfl {
			//fmt.Printf("%d durations with %d accesses.\n", len(durs),k)
			copy(recds[p:], durs)
			p += len(durs)
		}
		if p < len(recds) {
			fmt.Fprintln(of, "Something wrong here.")
		}*/
		sort.Sort(durarr(recds))
		fmt.Fprintf(of, "Median reconf-latency is %v\n.", recds[len(recds)/2])
		fmt.Fprintf(of, "Reconf-latency 95perc is %v\n.", recds[(len(recds)*19)/20])

	}

	if len(tupute) > 0 {

		tupute = combineTPut(tupute)
		fmt.Fprintf(of, "Throuputs: %d measurepoints.", len(tupute))
		PrintTputsAndReconfs(tupute, reconfe, of)
	}

	readc, _, _ := sortEvents(eventsperc)

	if len(readc) > 0 {
		maxlats := make([]time.Duration, 0, len(readc))
		cumovers := make([]time.Duration, 0, len(readc))

		for _, rc := range readc {
			o, m := CumOverAndMax(rc, *norm, normlat)
			if m > time.Duration(0) {
				maxlats = append(maxlats, m)
				cumovers = append(cumovers, o)
			}
		}
		fmt.Fprintf(of, "%d readclients\n", len(maxlats))
		fmt.Fprintf(of, "Average max-latency is %v.\n", MeanDuration(maxlats...))
		fmt.Fprintf(of, "Average overhead is %v.\n", MeanDuration(cumovers...))

		if len(maxlats) > 10 {
			sort.Sort(durarr(maxlats))
			sort.Sort(durarr(cumovers))
			fmt.Fprintf(of, "Median max-latency is %v\n.", maxlats[len(maxlats)/2])
			fmt.Fprintf(of, "Max-latency 95perc is %v\n.", maxlats[(len(maxlats)*19)/20])
			fmt.Fprintf(of, "Median overhead is %v\n.", cumovers[len(cumovers)/2])
			fmt.Fprintf(of, "Overhead 95perc is %v\n.", cumovers[(len(cumovers)*19)/20])
		}
	}

}

func sortEvents(eventsperc [][]e.Event) (reade, writee, reconfe [][]e.Event) {
	reade = make([][]e.Event, 0, len(eventsperc))
	writee = make([][]e.Event, 0, 1)
	reconfe = make([][]e.Event, 0, 1)

	for _, clevs := range eventsperc {
		if len(clevs) == 0 {
			continue
		}
		switch clevs[0].Type {
		case e.ClientReadLatency:
			reade = append(reade, clevs)
		case e.ClientWriteLatency:
			writee = append(writee, clevs)
		case e.ClientReconfLatency:
			reconfe = append(reconfe, clevs)
		case e.ThroughputSample:
			fmt.Printf("Found throughput sample, stop handling events.")
			return
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
		if evt.EndTime.Sub(evt.Time) > 1000*time.Millisecond {
			fmt.Printf("Spike event %v.\n", evt)
			//continue
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
	dmap[uint64(1000)] = make([]time.Duration, 0, len(events))
	for _, evt := range events {
		if dmap[evt.Value] == nil {
			dmap[evt.Value] = []time.Duration{evt.EndTime.Sub(evt.Time)}
		} else {
			dmap[evt.Value] = append(dmap[evt.Value], evt.EndTime.Sub(evt.Time))
		}
		dmap[1000] = append(dmap[1000], evt.EndTime.Sub(evt.Time))
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

	readTP := make([]uint64, 0, 100)
	recTP := make([]uint64, 0, 100)

	out, err := os.Create("TPutTable")
	if err != nil {
		fmt.Println("Could not create file: TPutTable")
		return
	}
	defer out.Close()

	i := 0
	cnt := 0
	for k, tput := range tpute {
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

		if k < 1 || tput.Time.Sub(tpute[k-1].Time) < 800*time.Millisecond || tput.Time.Sub(tpute[k-1].Time) > 1200*time.Millisecond {
			continue
		}
		if count == 0 {
			if cnt == 0 && len(recTP) > 0 {
				readTP = readTP[:len(readTP)-1]
				recTP = recTP[:len(recTP)-1]
				cnt = 3
			}
		}
		if cnt > 0 {
			cnt--
		} else {
			readTP = append(readTP, tput.Value)
			recTP = append(recTP, uint64(count))
		}

		fmt.Fprintf(out, "%d,%d\n", count, tput.Value)
		fmt.Fprintf(of, "Initialized %d reconfigurations before: ", count)
		fmt.Fprintf(of, "%v\n", tput)

	}
	fmt.Fprintf(of, "Mean Read TP: %d; Mean Reconf TP %d\n", mean64(readTP), mean64(recTP))
}

func CumOverAndMax(evts []e.Event, normal int, normlat time.Duration) (cumOver, maxlat time.Duration) {
	for _, ev := range evts {
		if ev.Type == e.ThroughputSample {
			continue
		}
		//if i > 0 {
		/*if ev.Type != evts[i-1].Type {
				fmt.Println("Different types submitted to CumOverMax")
				return
			}
		}*/
		if int(ev.Value) == normal {
			continue
		}
		dur := ev.EndTime.Sub(ev.Time)
		if dur > 5000*time.Millisecond {
			fmt.Printf("Spike latency %v at time %v with %d accesses\n", dur, ev.Time, ev.Value)
		} else {
			if dur > maxlat {
				maxlat = dur
			}
			cumOver += dur - normlat
		}
	}
	return
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
