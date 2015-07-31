package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	e "github.com/relab/smartMerge/elog/event"
)

func main() {
	var file = flag.String("file", "", "elog files to parse, separated by comma")
	//var filter = flag.Bool("filter", true, "filter out throughput samples")
	var outfile = flag.String("outfile", "", "write results to file")
	var list = flag.Bool("list", false, "print a list or latencies")

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
		fl, err := os.OpenFile(*outfile,os.O_APPEND|os.O_WRONLY,0666)
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
	events := make([]e.Event,0,len(infiles))

	for _,fi := range infiles {
		if fi == "" {
			continue
		}
		fievents, err := e.Parse(fi)
		if err != nil {
			fmt.Printf("Error %v  parsing events from %v", err, fi)
			return
		}
		for _,e := range fievents {
			events = append(events, e)
		}
	}

	if *list {
		fmt.Printf("Type of first event: %v", events[0].Type)
		for _,evt := range events {
			_, err := fmt.Fprintf(of, "%d: %d\n", evt.Value, evt.EndTime.Sub(evt.Time))
			if err != nil {
				fmt.Println("Error writing to file:", err)
			}
		}
		return
	}

	readl, writel, reconfl := sortLatencies(events)
	if len(readl) > 0 {
		_,err := fmt.Fprintln(of, "Read Latencies:")
		if err != nil {
			fmt.Println("Error writing to file:", err)
		}
		for k, durs := range readl {
			avg := MeanDuration(durs...)
			fmt.Fprintf(of, "Accesses %2d, %5d times, AvgLatency: %v\n", k, len(durs), avg)
		}
	}
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
			if k != 2 {
				affected += time.Duration(len(durs))
				totalaffected += time.Duration(len(durs)) * avgWrites[k]
			}
		}
		if affected > 0 {
			fmt.Fprintf(of,"Average latency for writes affected by reconfiguration: %v\n", (totalaffected/affected) )
			fmt.Fprintf(of, "Total overhead is: %v",totalaffected - (affected * avgWrites[2]) )
		}
		
	}

	if len(reconfl) > 0 {
		var total time.Duration
		var number time.Duration
		_,err := fmt.Fprintln(of, "Reconf Latencies:")
		if err != nil {
			fmt.Println("Error writing to file:", err)
		}
		for k, durs := range reconfl {
			avg := MeanDuration(durs...)
			fmt.Fprintf(of, "Accesses %2d, %5d times, AvgLatency: %v\n", k, len(durs), avg)
			total += avg * time.Duration(len(durs))
			number += time.Duration(len(durs))
		}
		fmt.Fprintf(of, "Average reconfiguration latency: %v", (total/number))
		fmt.Fprintf(of, "In total: %d reconfigurations", number)
	}
}

//Sort read, write and reconf latencies.
func sortLatencies(events []e.Event) (readl map[uint64][]time.Duration, writel map[uint64][]time.Duration, reconfl map[uint64][]time.Duration) {
	readl = make(map[uint64][]time.Duration, 0)
	writel = make(map[uint64][]time.Duration, 0)
	reconfl = make(map[uint64][]time.Duration, 0)
	for _, evt := range events {
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
