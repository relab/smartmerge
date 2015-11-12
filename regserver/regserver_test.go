package regserver

import (
	"encoding/binary"
	"fmt"
	"testing"

	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
	//"google.golang.org/grpc"
)

var ctx = context.Background()
var bytes = make([]byte, 64)

var one = uint32(1)
var two = uint32(2)
var tre = uint32(3)

var n11 = &pb.Node{one, one}
var n12 = &pb.Node{one, two}
var n22 = &pb.Node{two, two}
var n32 = &pb.Node{tre, two}
var n33 = &pb.Node{tre, tre}

var b1 = &pb.Blueprint{[]*pb.Node{n11}, one, one}
var b2 = &pb.Blueprint{[]*pb.Node{n22}, two, one}
var b12 = &pb.Blueprint{[]*pb.Node{n11,n22}, two, one}
var b22 = &pb.Blueprint{[]*pb.Node{n11,n22}, two, two}
var b23 = &pb.Blueprint{[]*pb.Node{n11,n22}, tre, two}

var b12x = &pb.Blueprint{[]*pb.Node{n12,n22}, two, one}
var b123 = &pb.Blueprint{[]*pb.Node{n12,n22,n32}, two,one}
var bx = &pb.Blueprint{[]*pb.Node{n11,n33},tre, two}
var by = &pb.Blueprint{[]*pb.Node{n12,n32},two, one}
var b0 *pb.Blueprint



func Put(x int, bytes []byte) []byte {
	binary.PutUvarint(bytes, uint64(x))
	return bytes
}

func Get(bytes []byte) int {
	x, _ := binary.Uvarint(bytes)
	return int(x)
}

func TestWriteAWriteS(t *testing.T) {
	rs := NewRegServer(false)
	var bytes = make([]byte, 64)
	bytes = Put(5, bytes)
	s := &pb.State{Value: bytes,Timestamp: 2,Writer: 0}

	// Test it returns no error and writes
	stest, err := rs.AWriteS(ctx, &pb.WriteS{State: s})
	if err != nil {
		t.Error("Did return error")
	}
	if rs.RState != s {
		t.Error("did not write")
	}

	s0 := &pb.State{Value: nil,Timestamp: 1,Writer: 0}
	rs.Cur = b2
	rs.CurC = uint32(b2.Len())

	//Can abort
	stest, _ = rs.AWriteS(ctx, &pb.WriteS{State: s0, Conf: &pb.Conf{one, one}})
	if rs.RState == s0 {
		t.Error("did write value with smaller timestamp")
	}
	if !stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Error("writeS did return correct abort")
	}

	//Does not abort, but sends cur, and new state.
	s2 := &pb.State{Value: nil,Timestamp: 2,Writer: 1}
	stest, _ = rs.AWriteS(ctx, &pb.WriteS{State: s2, Conf: &pb.Conf{Cur: one, This: uint32(b2.Len())}})
	if stest.Cur.Abort || stest.Cur.Cur != b2  {
		t.Error("writeS did not return correct cur.")
	}
	if rs.RState != s2 {
		t.Error("writeS did not write")
	}

	// If noabort is true, does not abort, but sends cur, state and next.
	s3 := &pb.State{Value: nil,Timestamp: 3,Writer: 0}
	rs.noabort = true
	rs.Next = []*pb.Blueprint{b12, b12x}
	stest, _ = rs.AWriteS(ctx, &pb.WriteS{State: s3, Conf: &pb.Conf{Cur: one, This: one}})
	if stest.Cur.Abort || stest.Cur.Cur != b2  {
		t.Error("writeS did not return correct cur.")
	}
	if rs.RState != s3 {
		t.Error("writeS returned wrong state")
	}
	if len(stest.Next) != 2 {
		t.Error("writeS did not return correct Next")
	}

	// Only send next that is large.
	stest, _ = rs.AWriteS(ctx, &pb.WriteS{Conf: &pb.Conf{uint32(b12.Len()), uint32(b12.Len())}})
	if stest.Cur != nil  {
		t.Error("writeS did not return correct cur.")
	}
	if len(stest.Next) != 1 {
		t.Error("writeS did not return correct Next")
	}
}

func TestWriteAReadS(t *testing.T) {
	rs := NewRegServer(false)
	var bytes = make([]byte, 64)
	bytes = Put(5, bytes)
	s := &pb.State{bytes, 2, 0}

	// Test it returns no error
	stest, err := rs.AReadS(ctx, &pb.Conf{})
	if err != nil {
		t.Error("Did return error")
	}
	fmt.Printf("Direct ReadS returned: %v\n", stest)
	fmt.Printf("Should return: %v\n", &InitState)

	rs.RState = s
	rs.Cur = b2
	rs.CurC = uint32(b2.Len())

	//Can abort
	stest, _ = rs.AReadS(ctx, &pb.Conf{one, one})
	if !stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Error("read S did return correct abort")
	}

	//Does not abort, but sends cur, and new state.
	stest, _ = rs.AReadS(ctx, &pb.Conf{Cur: one, This: uint32(b2.Len())})
	if stest.Cur.Abort || stest.Cur.Cur != b2  {
		t.Error("read S did not return correct cur.")
	}
	if stest.State.Compare(s) != 0 {
		t.Error("readS returned wrong state")
	}

	// If noabort is true, does not abort, but sends cur, state and next.
	rs.noabort = true
	rs.Next = []*pb.Blueprint{b12, b12x}
	stest, _ = rs.AReadS(ctx, &pb.Conf{one, one})
	if stest.Cur.Abort || stest.Cur.Cur != b2  {
		t.Error("read S did not return correct cur.")
	}
	if stest.State.Compare(s) != 0 {
		t.Error("readS returned wrong state")
	}
	if len(stest.Next) != 2 {
		t.Error("readS did not return correct Next")
	}

	// Only send next that is large.
	stest, _ = rs.AReadS(ctx, &pb.Conf{uint32(b12.Len()), uint32(b12.Len())})
	if stest.Cur != nil  {
		t.Error("read S did not return correct cur.")
	}
	if stest.State.Compare(s) != 0 {
		t.Error("readS returned wrong state")
	}
	if len(stest.Next) != 1 {
		t.Error("readS did not return correct Next")
	}


}
/*


	rs.AWriteS(ctx, &pb.WriteRequest{State: s})
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
			if lat.Equals(ab, bl) {
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
/*
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
			if lat.Equals(ab, bl) {
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
*/
