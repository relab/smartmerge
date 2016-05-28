package event

import (
	"fmt"
	"time"
)

type Event struct {
	Type    Type
	Time    time.Time
	EndTime time.Time
	Value   uint64
}

type Type uint8

const (
	// General: 0-15
	Unknown       Type = 0
	Start         Type = 1
	Running       Type = 2
	Processing    Type = 3
	ShutdownStart Type = 4
	Exit          Type = 5

	// Throughput: 16-23
	ThroughputSample Type = 16

	// Client Request Latency: 88-95
	ClientReadLatency   Type = 88
	ClientWriteLatency  Type = 89
	ClientReconfLatency Type = 90
)

//go:generate stringer -type=Type

func NewEvent(t Type) Event {
	return Event{
		Type: t,
		Time: time.Now(),
	}
}

func NewEventWithMetric(t Type, v uint64) Event {
	return Event{
		Type:  t,
		Time:  time.Now(),
		Value: v,
	}
}

func NewTimedEvent(t Type, start time.Time) Event {
	return Event{
		Type:    t,
		Time:    start,
		EndTime: time.Now(),
	}
}

func NewTimedEventWithMetric(t Type, start time.Time, v uint64) Event {
	return Event{
		Type:    t,
		Time:    start,
		EndTime: time.Now(),
		Value:   v,
	}
}

const layout = "2006-01-02 15:04:05.999999999"

func (e Event) String() string {
	switch e.Type {
	case ThroughputSample:
		return fmt.Sprintf("%v:\t%30v %3d",
			e.Time.Format(layout), e.Type, e.Value)
	case ClientReadLatency, ClientWriteLatency, ClientReconfLatency:
		return fmt.Sprintf("%v:\t%30v Accesses: %2d, Latency: %v",
			e.EndTime.Format(layout), e.Type, e.Value, e.EndTime.Sub(e.Time))
	default:
		if e.EndTime.IsZero() {
			return fmt.Sprintf("%v:\t%30v",
				e.Time.Format(layout), e.Type)
		}
		return fmt.Sprintf("%v:\t%30v %v",
			e.Time.Format(layout), e.Type, e.EndTime.Format(layout))
	}
}
