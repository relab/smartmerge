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

	for _, fi := range infiles {
		if fi == "" {
			continue
		}
		fievents, err := e.Parse(fi)
		if err != nil {
			fmt.Printf("Error %v  parsing events from %v", err, fi)
			return
		}
		for _, e := range fievents {
			events = append(events, e)
		}
	}

	if *debug {
		fmt.Fprintf(of, "%v\n", events[0])
		cnt := 0
		for _, evt := range events {
			if evt.EndTime.Sub(evt.Time) > 100*time.Millisecond {
				fmt.Fprintf(of, "%v\n", evt)
				cnt++
			}
		}
		fmt.Fprintf(of, "%v\n", events[len(events)-1])
		fmt.Fprintf(of, "%d spike latencies.\n", cnt)
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

	readl, writel, reconfl := sortLatencies(events)
	if len(readl) > 0 {
		_, err := fmt.Fprintln(of, "Read Latencies:")
		if err != nil {
			fmt.Println("Error writing to file:", err)
		}
		for k, durs := range readl {
			avg := MeanDuration(durs...)
			fmt.Fprintf(of, "Accesses %2d, %5d times, AvgLatency: %v\n", k, len(durs), avg)
		}
	}
	normal := uint64(*norm)
	if len(writel) > 0 {
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
		}
	}

	if len(reconfl) > 0 {
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
}

//Sort read, write and reconf latencies.
func sortLatencies(events []e.Event) (readl map[uint64][]time.Duration, writel map[uint64][]time.Duration, reconfl map[uint64][]time.Duration) {
	readl = make(map[uint64][]time.Duration, 0)
	writel = make(map[uint64][]time.Duration, 0)
	reconfl = make(map[uint64][]time.Duration, 0)
	for _, evt := range events {
		if evt.EndTime.Sub(evt.Time) > 100 *time.Millisecond {
			fmt.Printf("Discarding event %v.\n", evt)
			continue
		}
		switch evt.Type {
		case e.ClientReadLatency:
			if readl[evt.Value] == nil {
				readl[evt.Value] = []time.Duration{evt.EndTime.Sub(evt.Time)}
			} else {
				readl[evt.Value] = append(readl[evt.Value], evt.EndTime.Sub(evt.Time))
			}
		case e.ClientWriteLatency:
			if writel[evt.Value] == nil {
				writel[evt.Value] = []time.Duration{evt.EndTime.Sub(evt.Time)}
			} else {
				writel[evt.Value] = append(writel[evt.Value], evt.EndTime.Sub(evt.Time))
			}
		case e.ClientReconfLatency:
			if reconfl[evt.Value] == nil {
				reconfl[evt.Value] = []time.Duration{evt.EndTime.Sub(evt.Time)}
			} else {
				reconfl[evt.Value] = append(reconfl[evt.Value], evt.EndTime.Sub(evt.Time))
			}
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
