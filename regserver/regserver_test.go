package regserver

import (
	"encoding/binary"
	"testing"
	
	pb "github.com/relab/smartMerge/proto"
	"golang.org/x/net/context"
	lat "github.com/relab/smartMerge/directCombineLattice"
)

var ctx = context.Background()
var bytes = make([]byte,64)

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
	x,_ := binary.Uvarint(bytes)
	return int(x)
}

func TestWriteReadS(t *testing.T) {
	rs := NewRegServer()
	var bytes = make([]byte,64)
	bytes = Put(5,bytes)
	s := &pb.State{bytes,2}
	
	rr,_ := rs.WriteS(ctx, s)
	if !rr.New {
		t.Error("Writing to initial state was not acknowledged.")
	}
	
	s = &pb.State{bytes,1}
	rr,_ = rs.WriteS(ctx, s)
	if rr.New {
		t.Error("Writing old value was acknowledged.")
	}
	
	s,_ = rs.ReadS(ctx, &pb.ReadRequest{})
	if s.Timestamp != int64(2) {
		t.Error("Reading returned wrong timestamp.")
	}
	if Get(bytes) != 5 {
		t.Error("Reading returned wrong bytes.")
	} 
	
	rs.WriteN(ctx, &bpi1)
	rs.WriteN(ctx, &bpi2)
	rs.WriteN(ctx, &bpi1)

	next,_ := rs.ReadN(ctx, &pb.ReadNRequest{})
	expected := []*pb.Blueprint{&bpi1, &bpi2}
	for _,ab := range next.Next {
		for i,bl := range expected {
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
	if len(next.Next) != 2 {
			t.Error("Some too many blueprints returned.")
	}
	
	if len(expected) != 0 {
			t.Error("Some expected blueprint was not returned.")
	}
}