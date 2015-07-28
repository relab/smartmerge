package event

import (
	"bytes"
	"encoding/gob"
	"io"
	"io/ioutil"
	
	"github.com/relab/smartMerge/elog"
)

func Parse(filename string) ([]Event, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(file)
	dec := gob.NewDecoder(buf)
	var events []Event

	for {
		var event Event
		err = dec.Decode(&event)
		if err != nil {
			if err == io.EOF {
				break
			}
			return events, err
		}
		events = append(events, event)
	}

	return events, nil
}

func ExtractThroughput(events []Event) (regular, throughput []Event) {
	regular = make([]Event, 0)
	throughput = make([]Event, 0)
	for _, event := range events {
		if event.Type == ThroughputSample {
			throughput = append(throughput, event)
		} else {
			regular = append(regular, event)
		}
	}

	return
}

func Combine(filename1, filename2 string) (error){
	events1, err := Parse(filename1)
	if err != nil {
		return err
	}
	events2, err := Parse(filename2)
	if err != nil {
		return err
	}
	
	elog.Enable()
	defer elog.Flush()
	for _,e := range events1 {
		elog.Log(e)
	}
	for _,e := range events2 {
		elog.Log(e)
	}
	
	
}