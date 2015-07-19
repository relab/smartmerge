package rpc

import (
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
)

var defaultLocMapper = func(ip net.IP) string {
	if ip.IsLoopback() {
		return "localhost"
	}
	return "unknown"
}

type Manager struct {
	ids      []uint32
	idsByLoc map[string][]uint32
	locs     []string
	machines map[uint32]*machine
	configs  map[uint32]*Configuration
	mu       sync.Mutex

	opts managerOptions
}

func NewManager(machines []string, opts ...ManagerOption) (*Manager, error) {
	if len(machines) == 0 {
		return nil, errors.New("could not create manager: no machines provided")
	}

	m := new(Manager)
	m.ids = make([]uint32, len(machines))
	m.machines = make(map[uint32]*machine)
	m.configs = make(map[uint32]*Configuration)

	m.idsByLoc = make(map[string][]uint32)

	for _, opt := range opts {
		opt(&m.opts)
	}

	if m.opts.locationMapper == nil {
		m.opts.locationMapper = defaultLocMapper
	}

	err := m.genMachines(machines)
	if err != nil {
		return nil, fmt.Errorf("could not create manager: %v", err)
	}

	err = m.connectAll(m.opts.grpcDialOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not create manager: %v", err)
	}

	m.locs = make([]string, 0)
	for loc := range m.idsByLoc {
		m.locs = append(m.locs, loc)
	}
	sort.Sort(sort.StringSlice(m.locs))

	return m, nil
}

func (m *Manager) IDs() []uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ids
}

func (m *Manager) IDsByLocation(location string) []uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.idsByLoc[location]
}

func (m *Manager) Locations() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.locs
}

func (m *Manager) Machines() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ma := make([]string, len(m.ids))
	for i, id := range m.ids {
		ma[i] = m.machines[id].String()
	}
	return ma
}

func (m *Manager) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.ids)
}

func (m *Manager) genMachines(machineNames []string) error {
	h := fnv.New32a()
	for i, mn := range machineNames {
		tcpAddr, err := net.ResolveTCPAddr("tcp", mn)
		if err != nil {
			return fmt.Errorf("could not resolve tcp address %s: %v", mn, err)
		}
		loc := m.opts.locationMapper(tcpAddr.IP)
		h.Write([]byte(mn))
		id := h.Sum32()
		m.machines[id] = &machine{
			id:       id,
			rawAddr:  mn,
			tcpAddr:  tcpAddr,
			location: loc,
			latency:  time.Second * 30,
		}
		m.ids[i] = id
		locIDs, _ := m.idsByLoc[loc]
		locIDs = append(locIDs, id)
		m.idsByLoc[loc] = locIDs
		h.Reset()
	}
	return nil
}

func (m *Manager) connectAll(dialOpts ...grpc.DialOption) error {
	for id, ma := range m.machines {
		conn, err := grpc.Dial(ma.tcpAddr.String(), dialOpts...)
		if err != nil {
			return fmt.Errorf("dialing node %d failed: %v", id, err)
		}
		ma.conn = conn
	}
	return nil
}

func (m *Manager) NewConfiguration(ids []uint32, quorumSize int, grpcOptions ...grpc.CallOption) (*Configuration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(ids) == 0 {
		return nil, errors.New("need at least one machine in a configuration")
	}
	if quorumSize > len(ids) || quorumSize < 1 {
		return nil, fmt.Errorf("illegal quourm size (%d) for configuration size (%d)", quorumSize, len(ids))
	}

	h := fnv.New32a()
	h.Write([]byte(strconv.Itoa(quorumSize)))
	for _, id := range ids {
		_, found := m.machines[id]
		if !found {
			return nil, MachineNotFound(id)
		}
		h.Write([]byte(strconv.Itoa(int(id))))
	}
	cid := h.Sum32()
	c, found := m.configs[cid]
	if found {
		return c, nil
	}
	c = &Configuration{
		id:              cid,
		machines:        ids,
		quorum:          quorumSize,
		grpcCallOptions: grpcOptions,
		mgr:             m,
	}
	m.configs[cid] = c

	return c, nil
}

func (m *Manager) invoke(configID uint32, ctx context.Context, method string, args interface{}, opts ...grpc.CallOption) ([]rpcReply, error) {
	c, found := m.configs[configID]
	if !found {
		return nil, ConfigNotFound(configID)
	}

	var (
		replyChan  = make(chan rpcReply, c.quorum)
		stopSignal = make(chan struct{})
		errSignal  = make(chan bool, c.quorum)
		out        = make([]rpcReply, c.quorum)
		errCount   int
	)

	for _, mid := range c.machines {
		ma, found := m.machines[mid]
		if !found {
			panic("machine not found")
		}
		go func(machine *machine) {
			r := rpcReply{mid: machine.id}
			ce := make(chan error, 1)
			start := time.Now()
			go func() {
				ce <- grpc.Invoke(ctx, method, args, r.reply, machine.conn, c.grpcCallOptions...)
			}()
			select {
			case err := <-ce:
				if err != nil {
					machine.lastErr = err
					errSignal <- true
					return
				}
				machine.latency = time.Since(start)
				replyChan <- r
			case <-stopSignal:
				return
			}
		}(ma)
	}

	for {
		select {
		case r := <-replyChan:
			out = append(out, r)
			if len(out) >= c.quorum {
				close(stopSignal)
				return out, nil
			}
		case <-errSignal:
			errCount++
			if errCount > len(c.machines)-c.quorum {
				close(stopSignal)
				return nil, errors.New("could not complete request due to too many errors")
			}
		}
	}
}

type rpcReply struct {
	mid   uint32
	reply interface{}
}

func MachineID(IP string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(IP))
	return h.Sum32()
}

func (m *Manager) NewConfigurationFromIP(IPs []string, quorumSize int, grpcOptions ...grpc.CallOption) (*Configuration, error) {
	ids := make([]uint32, len(IPs))

	h := fnv.New32a()
	for i, IP := range IPs {
		h.Write([]byte(IP))
		ids[i] = h.Sum32()
		h.Reset()
	}

	c, err := m.NewConfiguration(ids, quorumSize, grpcOptions...)

	return c, err
}
