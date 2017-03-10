package regserver

import (
	"encoding/binary"
	"testing"

	bp "github.com/relab/smartmerge/blueprints"
	pb "github.com/relab/smartmerge/proto"
	"golang.org/x/net/context"
	//"google.golang.org/grpc"
)

var ctx = context.Background()
var bytes = make([]byte, 64)

var one = uint32(1)
var two = uint32(2)
var tre = uint32(3)

var n11 = &bp.Node{Id: one, Version: one}
var n12 = &bp.Node{Id: one, Version: two}
var n22 = &bp.Node{Id: two, Version: two}
var n32 = &bp.Node{Id: tre, Version: two}
var n33 = &bp.Node{Id: tre, Version: tre}

var b1 = &bp.Blueprint{Nodes: []*bp.Node{n11}, FaultTolerance: one, Epoch: one}
var b2 = &bp.Blueprint{Nodes: []*bp.Node{n22}, FaultTolerance: two, Epoch: one}
var b12 = &bp.Blueprint{Nodes: []*bp.Node{n11, n22}, FaultTolerance: two, Epoch: one}
var b22 = &bp.Blueprint{Nodes: []*bp.Node{n11, n22}, FaultTolerance: two, Epoch: two}
var b23 = &bp.Blueprint{Nodes: []*bp.Node{n11, n22}, FaultTolerance: tre, Epoch: two}

var b12x = &bp.Blueprint{Nodes: []*bp.Node{n12, n22}, FaultTolerance: two, Epoch: one}
var b123 = &bp.Blueprint{Nodes: []*bp.Node{n12, n22, n32}, FaultTolerance: two, Epoch: one}
var bx = &bp.Blueprint{Nodes: []*bp.Node{n11, n33}, FaultTolerance: tre, Epoch: two}
var by = &bp.Blueprint{Nodes: []*bp.Node{n12, n32}, FaultTolerance: two, Epoch: one}
var b0 *bp.Blueprint

func Put(x int, bytes []byte) []byte {
	binary.PutUvarint(bytes, uint64(x))
	return bytes
}

func Get(bytes []byte) int {
	x, _ := binary.Uvarint(bytes)
	return int(x)
}

func TestSetState(t *testing.T) {
	rs := NewRegServer(false)
	rs.Next = []*bp.Blueprint{b12, b12x}
	//Perfectly normal SetState
	stest, err := rs.SetState(ctx, &pb.NewState{
		//Cur:     b2,
		CurC:    uint32(b2.Len()),
		State:   &pb.State{Value: nil, Timestamp: 2, Writer: 0},
		LAState: b1,
	})
	if err != nil || rs.RState.Compare(&pb.State{Value: nil, Timestamp: 2, Writer: 0}) != 0 || !rs.LAState.Equals(b1) {
		t.Error("first write did not work.")
	}
	if len(stest.Next) != 2 {
		t.Error("did not return correct next")
	}

	// Set state in Cur.
	stest, _ = rs.SetState(ctx, &pb.NewState{
		//Cur:     b2,
		CurC:    uint32(b2.Len()),
		State:   &pb.State{Value: nil, Timestamp: 2, Writer: 1},
		LAState: b2,
	})
	if rs.RState.Compare(&pb.State{Value: nil, Timestamp: 2, Writer: 1}) != 0 || !rs.LAState.Equals(b12) {
		t.Error("did not set state correctly")
	}
	if len(stest.Next) != 2 {
		t.Error("did not return correct next")
	}
	if stest.Cur != nil {
		t.Error("did return wrong cur")
	}

	// Clean next on set state
	stest, _ = rs.SetState(ctx, &pb.NewState{
		//Cur:     b12,
		CurC:    uint32(b12.Len()),
		LAState: b12x,
	})
	if rs.RState.Compare(&pb.State{Value: nil, Timestamp: 2, Writer: 1}) != 0 || !rs.LAState.Equals(b12x) {
		t.Error("did not set state correctly")
	}
	if len(rs.Next) != 2 {
		t.Error("did not clean up Next, b12")
	}
	if len(stest.Next) != 1 {
		t.Error("did not return correct next")
	}
	if stest.Cur != nil {
		t.Error("did return wrong cur")
	}

	rs.Cur = b12.Copy()
	rs.CurC = uint32(b12.Len())

	// Set state in old cur
	stest, _ = rs.SetState(ctx, &pb.NewState{
		//Cur:     b2,
		CurC:    uint32(b2.Len()),
		State:   &pb.State{Value: nil, Timestamp: 3, Writer: 0},
		LAState: b123,
	})
	if rs.RState.Compare(&pb.State{Value: nil, Timestamp: 3, Writer: 0}) != 0 || !rs.LAState.Equals(b123) {
		t.Error("did not set state correctly")
	}
	if len(rs.Next) != 2 {
		t.Error("did not clean up Next")
	}
	if !stest.Cur.Equals(b12) {
		t.Errorf("did return wrong cur, %v instead of %v", stest.Cur, b12)
	}
}

func TestLAProp(t *testing.T) {
	rs := NewRegServer(false)
	var bytes = make([]byte, 64)
	bytes = Put(5, bytes)
	rs.Next = []*bp.Blueprint{b12}

	// Test it returns no error and writes
	stest, err := rs.LAProp(ctx, &pb.LAProposal{Prop: b12, Conf: &pb.Conf{}})
	if err != nil {
		t.Error("Did return error")
	}
	if rs.LAState != b12 {
		t.Error("did not write")
	}
	if stest.LAState != nil {
		t.Error("did return LAState")
	}

	rs.Cur = b2
	rs.CurC = uint32(b2.Len())

	//Can abort
	stest, _ = rs.LAProp(ctx, &pb.LAProposal{Prop: b12x, Conf: &pb.Conf{This: one, Cur: one}})
	if rs.LAState != b12 {
		t.Error("did write on abort")
	}
	if !stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Error("laprop did return correct abort")
	}

	//Does not abort, but return cur, does not write old value.
	stest, _ = rs.LAProp(ctx, &pb.LAProposal{Prop: b2, Conf: &pb.Conf{This: uint32(b2.Len()), Cur: uint32(b2.Len())}})
	if stest.Cur.Abort {
		t.Errorf("laprop did not return correct cur, Abort was %v, Cur was %v.", stest.Cur.Abort, stest.Cur.Cur)
	}
	if !stest.LAState.Equals(b12) {
		//fmt.Println(stest.LAState)
		t.Error("did not return LAState")
	}
	if !rs.LAState.Equals(b12) {
		t.Error("wrong state")
	}

	// If noabort is true, does not abort, but sends cur, state and next.
	rs.Next = []*bp.Blueprint{b12, b12x}
	rs.noabort = true
	stest, _ = rs.LAProp(ctx, &pb.LAProposal{Prop: by, Conf: &pb.Conf{Cur: one, This: one}})
	if stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Error("laprop did not return correct cur.")
	}
	if !rs.LAState.Equals(b123) {
		t.Error("laprop did not write correctly")
	}
	if !stest.LAState.Equals(b123) {
		t.Error("did not return LAState")
	}

	// Only send next that is large.
	stest, _ = rs.LAProp(ctx, &pb.LAProposal{Prop: bx, Conf: &pb.Conf{Cur: uint32(b12.Len()), This: uint32(b12.Len())}})
	if stest.Cur.Cur != nil || stest.Cur.Abort {
		t.Errorf("laprop did not return correct cur, did get %v expecting nil.", stest.Cur)
	}
	if !rs.LAState.Equals(bx.Merge(b123)) {
		t.Error("laprop did not result in correct state.")
	}
	if !stest.LAState.Equals(bx.Merge(b123)) {
		t.Error("laprop did not return correct state.")
	}

}

func TestWriteNext(t *testing.T) {
	rs := NewRegServer(false)
	var bytes = make([]byte, 64)
	bytes = Put(5, bytes)
	s := &pb.State{Value: bytes, Timestamp: 2, Writer: 0}

	// Test it returns no error and writes
	stest, err := rs.WriteNext(ctx, &pb.WriteN{Next: b12})
	if err != nil {
		t.Error("Did return error")
	}
	if len(rs.Next) != 1 {
		t.Error("did not write")
	}

	rs.Cur = b2
	rs.CurC = uint32(b2.Len())
	rs.LAState = b12x
	rs.RState = s

	//Can abort
	stest, _ = rs.WriteNext(ctx, &pb.WriteN{Next: b12x, CurC: one})
	if len(rs.Next) != 1 {
		t.Error("did write next on abort")
	}
	if !stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Error("writeN did return correct abort")
	}

	//Does not abort, does not write duplicate next.
	stest, _ = rs.WriteNext(ctx, &pb.WriteN{Next: b12, CurC: uint32(b2.Len())})
	if stest.Cur.Cur != nil || stest.Cur.Abort {
		t.Errorf("writeN did not return correct cur, instead %v.", stest.Cur)
	}
	if stest.State != s {
		t.Error("writeN did not return state")
	}
	if len(rs.Next) != 1 {
		t.Error("did write duplicate next")
	}
	if stest.LAState != b12x {
		t.Error("did not return LAState")
	}
	if len(stest.GetCur().Next) != 1 {
		t.Error("writeN did not return correct next")
	}

	// If noabort is true, does not abort, but sends cur, state and next.
	rs.noabort = true
	stest, _ = rs.WriteNext(ctx, &pb.WriteN{Next: b12x, CurC: one})
	if stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Error("writeN did not return correct cur.")
	}
	if stest.State != s {
		t.Error("writeN returned wrong state")
	}
	if len(stest.Cur.Next) != 2 {
		t.Error("writeN did not return correct Next")
	}
	if len(rs.Next) != 2 {
		t.Error("writeN did not write correctly")
	}
	if stest.LAState != b12x {
		t.Error("did not return LAState")
	}

	// Only send next that is large.
	stest, _ = rs.WriteNext(ctx, &pb.WriteN{CurC: uint32(b12.Len())})
	if stest.Cur.Cur != nil || stest.Cur.Abort {
		t.Error("writeN did not return correct cur.")
	}
	if len(stest.Cur.Next) != 1 {
		t.Error("writeN did not return correct Next")
	}
}

func TestWriteWrite(t *testing.T) {
	rs := NewRegServer(false)
	var bytes = make([]byte, 64)
	bytes = Put(5, bytes)
	s := &pb.State{Value: bytes, Timestamp: 2, Writer: 0}

	// Test it returns no error and writes
	stest, err := rs.Write(ctx, &pb.WriteS{State: s})
	if err != nil {
		t.Error("Did return error")
	}
	if rs.RState != s {
		t.Error("did not write")
	}

	s0 := &pb.State{Value: nil, Timestamp: 1, Writer: 0}
	rs.Cur = b2
	rs.CurC = uint32(b2.Len())

	//Can abort
	stest, _ = rs.Write(ctx, &pb.WriteS{State: s0, Conf: &pb.Conf{Cur: one, This: one}})
	if rs.RState == s0 {
		t.Error("did write value with smaller timestamp")
	}
	if !stest.Abort || stest.Cur != b2 {
		t.Error("writeS did return correct abort")
	}

	//Does not abort, but sends cur, and new state.
	s2 := &pb.State{Value: nil, Timestamp: 2, Writer: 1}
	stest, _ = rs.Write(ctx, &pb.WriteS{State: s2, Conf: &pb.Conf{This: uint32(b2.Len()) + 1, Cur: uint32(b2.Len())}})
	if stest.Abort || stest.Cur != nil {
		t.Errorf("writeS did not return correct cur, instead %v, abort was %v.", stest.Cur, stest.Abort)
	}
	if rs.RState != s2 {
		t.Error("writeS did not write")
	}

	// If noabort is true, does not abort, but sends cur, state and next.
	s3 := &pb.State{Value: nil, Timestamp: 3, Writer: 0}
	rs.noabort = true
	rs.Next = []*bp.Blueprint{b12, b12x}
	stest, _ = rs.Write(ctx, &pb.WriteS{State: s3, Conf: &pb.Conf{Cur: one, This: one}})
	if stest.Abort || stest.Cur != b2 {
		t.Error("writeS did not return correct cur.")
	}
	if rs.RState != s3 {
		t.Error("writeS returned wrong state")
	}
	if len(stest.Next) != 2 {
		t.Error("writeS did not return correct Next")
	}

	// Only send next that is large.
	stest, _ = rs.Write(ctx, &pb.WriteS{Conf: &pb.Conf{Cur: uint32(b12.Len()), This: uint32(b12.Len())}})
	if stest.Cur != nil {
		t.Error("writeS did not return correct cur.")
	}
	if len(stest.Next) != 1 {
		t.Error("writeS did not return correct Next")
	}
}

func TestWriteRead(t *testing.T) {
	rs := NewRegServer(false)
	var bytes = make([]byte, 64)
	bytes = Put(5, bytes)
	s := &pb.State{Value: bytes, Timestamp: 2, Writer: 0}

	// Test it returns no error
	stest, err := rs.Read(ctx, &pb.Conf{})
	if err != nil {
		t.Error("Did return error")
	}
	if stest.State.Timestamp != 0 || stest.State.Writer != 0 {
		t.Errorf("Direct ReadS returned: %v\n Should return: %v", stest, &pb.State{Value: make([]byte, 0), Timestamp: int32(0), Writer: uint32(0)})
	}

	rs.RState = s
	rs.Cur = b2
	rs.CurC = uint32(b2.Len())

	//Can abort
	stest, _ = rs.Read(ctx, &pb.Conf{Cur: one, This: one})
	if !stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Error("read S did return correct abort")
	}

	//Does not abort, but sends cur, and new state.
	stest, _ = rs.Read(ctx, &pb.Conf{This: uint32(b2.Len()), Cur: uint32(b2.Len())})
	if stest.Cur != nil {
		//if stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Errorf("read S did not return correct cur, but.")
	}
	if stest.State.Compare(s) != 0 {
		t.Error("readS returned wrong state")
	}

	// If noabort is true, does not abort, but sends cur, state and next.
	rs.noabort = true
	rs.Next = []*bp.Blueprint{b12, b12x}
	stest, _ = rs.Read(ctx, &pb.Conf{Cur: one, This: one})
	if stest.Cur.Abort || stest.Cur.Cur != b2 {
		t.Error("read S did not return correct cur.")
	}
	if stest.State.Compare(s) != 0 {
		t.Error("readS returned wrong state")
	}
	if len(stest.Cur.Next) != 2 {
		t.Error("readS did not return correct Next")
	}

	// Only send next that is large.
	stest, _ = rs.Read(ctx, &pb.Conf{Cur: uint32(b12.Len()), This: uint32(b12.Len())})
	if stest.Cur.Cur != nil {
		t.Errorf("read S did not return correct cur, instead %v.", stest.Cur)
	}
	if stest.State.Compare(s) != 0 {
		t.Error("readS returned wrong state")
	}
	if len(stest.Cur.Next) != 1 {
		t.Error("readS did not return correct Next")
	}

}
