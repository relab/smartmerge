package regserver

import (
	"encoding/binary"
	"fmt"
	"testing"

	lat "github.com/relab/smartMerge/directCombineLattice"
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var ctx = context.Background()
var bytes = make([]byte, 64)

var onei = uint32(1)
var twoi = uint32(2)
var trei = uint32(3)

var a1i = []uint32{onei, twoi}
var r1i = []uint32{onei}
var r2i = []uint32{trei}
var bpi1 = pb.Blueprint{r1i, r2i}
var bpi2 = pb.Blueprint{a1i, r2i}

func Put(x int, bytes []byte) []byte {
	binary.PutUvarint(bytes, uint64(x))
	return bytes
}

func Get(bytes []byte) int {
	x, _ := binary.Uvarint(bytes)
	return int(x)
}

func TestWriteReadS(t *testing.T) {
	rs := NewRegServer()
	var bytes = make([]byte, 64)
	bytes = Put(5, bytes)
	s := &pb.State{bytes, 2, 0}

	stest, _ := rs.ReadS(ctx, &pb.ReadRequest{})
	fmt.Printf("Direct ReadS returned: %v\n", stest)
	fmt.Printf("Should return: %v\n", &InitState)

	rs.WriteS(ctx, &pb.WriteRequest{State: s})
	if rs.RState != s {
		t.Error("First write did fail.")
	}

	s2 := &pb.State{bytes, 1, 0}
	rs.WriteS(ctx, &pb.WriteRequest{State: s2})
	if rs.RState != s {
		t.Error("Second write did fail.")
	}

	rrep, _ := rs.ReadS(ctx, &pb.ReadRequest{})
	if rrep.State.Compare(s) != 0 {
		t.Error("Reading returned wrong timestamp.")
	}
	if Get(bytes) != 5 {
		t.Error("Reading returned wrong bytes.")
	}

	rs.WriteN(ctx, &pb.WriteNRequest{Next: &bpi1})
	rs.WriteN(ctx, &pb.WriteNRequest{Next: &bpi2})
	rs.WriteN(ctx, &pb.WriteNRequest{Next: &bpi1})

	rNrep, _ := rs.ReadN(ctx, &pb.ReadNRequest{})
	expected := []*pb.Blueprint{&bpi1, &bpi2}
	for _, ab := range rNrep.Next {
		for i, bl := range expected {
			if lat.Equals(*ab, *bl) {
				if i == 1 {
					expected = expected[:1]
				} else {
					expected = expected[1:]
				}
				break
			}
		}
	}
	if len(rNrep.Next) != 2 {
		t.Error("Some too many blueprints returned.")
	}

	if len(expected) != 0 {
		t.Error("Some expected blueprint was not returned.")
	}
}

func TestAdvWriteReadS(t *testing.T) {
	rs := NewRegServer()
	var bytes = make([]byte, 64)
	bytes = Put(5, bytes)
	s := &pb.State{bytes, 2, 0}

	stest, _ := rs.AReadS(ctx, &pb.AdvRead{})
	fmt.Printf("Direct ReadS returned: %v\n", stest)
	fmt.Printf("Should return: %v\n", &InitState)

	rs.AWriteS(ctx, &pb.AdvWriteS{State: s})
	if rs.RState != s {
		t.Error("First write did fail.")
	}

	s2 := &pb.State{bytes, 1, 0}
	rs.AWriteS(ctx, &pb.AdvWriteS{State: s2})
	if rs.RState != s {
		t.Error("Second write did fail.")
	}

	rrep, _ := rs.AReadS(ctx, &pb.AdvRead{})
	if rrep.State.Compare(s) != 0 {
		t.Error("Reading returned wrong timestamp.")
	}
	if Get(bytes) != 5 {
		t.Error("Reading returned wrong bytes.")
	}

	rs.AWriteN(ctx, &pb.AdvWriteN{Next: &bpi1})
	rs.AWriteN(ctx, &pb.AdvWriteN{Next: &bpi2})
	rs.AWriteN(ctx, &pb.AdvWriteN{Next: &bpi1})

	rNrep, _ := rs.AWriteS(ctx, &pb.AdvWriteS{State: s})
	expected := []*pb.Blueprint{&bpi1, &bpi2}
	for _, ab := range rNrep.Next {
		for i, bl := range expected {
			if lat.Equals(*ab, *bl) {
				if i == 1 {
					expected = expected[:1]
				} else {
					expected = expected[1:]
				}
				break
			}
		}
	}
	if len(rNrep.Next) != 2 {
		t.Error("Some too many blueprints returned.")
	}

	if len(expected) != 0 {
		t.Error("Some expected blueprint was not returned.")
	}
}

func TestStartStop(t *testing.T) {
	Start(10000)

	var opts []grpc.DialOption
	conn, err := grpc.Dial("127.0.0.1:10000", opts...)
	if err != nil {
		t.Errorf("fail to dial: %v", err)
	}
	defer conn.Close()

	cl := pb.NewRegisterClient(conn)

	s, err := cl.ReadS(ctx, &pb.ReadRequest{})
	if err != nil {
		t.Errorf("ReadS returned error: %v", err)
	}
	fmt.Printf("ReadS returned %v\n", s)

	err = Stop()
	if err != nil {
		t.Error("Stop returned error.")
	}
}

func TestAdvStartStop(t *testing.T) {
	StartAdv(10000)

	var opts []grpc.DialOption
	conn, err := grpc.Dial("127.0.0.1:10000", opts...)
	if err != nil {
		t.Errorf("fail to dial: %v", err)
	}
	defer conn.Close()

	cl := pb.NewAdvRegisterClient(conn)

	s, err := cl.AReadS(ctx, &pb.AdvRead{})
	if err != nil {
		t.Errorf("ReadS returned error: %v", err)
	}
	fmt.Printf("ReadS returned %v\n", s)

	err = Stop()
	if err != nil {
		t.Error("Stop returned error.")
	}
}

func TestStartTStop(t *testing.T) {
	srv, err := StartTest(10000)
	if err != nil {
		t.Errorf("fail to start: %v", err)
	}

	defer srv.Stop()
	var opts []grpc.DialOption
	conn, err := grpc.Dial("127.0.0.1:10000", opts...)
	if err != nil {
		t.Errorf("fail to dial: %v", err)
	}
	defer conn.Close()

	cl := pb.NewRegisterClient(conn)

	rSrep, err := cl.ReadS(ctx, &pb.ReadRequest{})
	if err != nil {
		t.Errorf("ReadS returned error: %v", err)
	}
	if rSrep.Cur != nil {
		t.Errorf("ReadS returned a new current conf: %v", rSrep.Cur)
	}

	cl.SetCur(ctx, &pb.NewCur{&bpi1, twoi})
	rSrep, err = cl.ReadS(ctx, &pb.ReadRequest{twoi})
	if err != nil {
		t.Errorf("ReadS returned error: %v", err)
	}
	if rSrep.Cur != nil {
		t.Errorf("ReadS returned a new current conf: %v", rSrep.Cur)
	}
	rSrep, err = cl.ReadS(ctx, &pb.ReadRequest{onei})
	if err != nil {
		t.Errorf("ReadS returned error: %v", err)
	}
	fmt.Println(rSrep.Cur)
	if rSrep.Cur == nil {
		t.Errorf("ReadS returned a new current conf: %v", rSrep.Cur)
	}
}
