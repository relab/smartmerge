package rpc

import (
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	pb "github.com/relab/smartMerge/proto"
)

type machine struct {
	id       uint32
	rawAddr  string
	tcpAddr  *net.TCPAddr
	location string
	conn     *grpc.ClientConn
	lastErr  error
	latency  time.Duration
	client	 pb.RegisterClient
}

func (m *machine) String() string {
	return fmt.Sprintf(
		"machine %d | addr: %s | location: %s | latency: %v",
		m.id,
		m.rawAddr,
		m.location,
		m.latency,
	)
}

type byID []machine

func (p byID) Len() int           { return len(p) }
func (p byID) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p byID) Less(i, j int) bool { return p[i].id < p[j].id }

type byLatency []machine

func (p byLatency) Len() int           { return len(p) }
func (p byLatency) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p byLatency) Less(i, j int) bool { return p[i].latency < p[j].latency }
