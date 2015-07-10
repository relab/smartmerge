package rpc

import (
	"fmt"
	"testing"
	pb "github.com/relab/smartMerge/proto"
	lat "github.com/relab/smartMerge/directCombineLattice"
)

var a1 = []uint32{1}
var a2 = []uint32{1,2}
var a3 = []uint32{1,2,3}
var r1 = []uint32{4}
var r2 = []uint32{4,5}
var r3 = []uint32{4,5,6}

var bp1 = pb.Blueprint{a1,r1}
var bp2 = pb.Blueprint{a2,r1}
var bp3 = pb.Blueprint{a2,r2}
var bp4 = pb.Blueprint{a3,r2}
var bp5 = pb.Blueprint{a3,r3}


func Setup() ([]*pb.ReadNReply, []lat.Blueprint) {
	replies := make([]*pb.ReadNReply,5)
	replies[0]=&pb.ReadNReply{[]*pb.Blueprint{&bp1,&bp3}}
	replies[1]=&pb.ReadNReply{[]*pb.Blueprint{&bp2, &bp3,&bp4}}
	replies[2]=&pb.ReadNReply{[]*pb.Blueprint{&bp2, &bp4, &bp5, &bp1}}
	replies[3]=&pb.ReadNReply{[]*pb.Blueprint{&bp1,&bp2, &bp3, &bp5}}
	replies[4]=&pb.ReadNReply{[]*pb.Blueprint{&bp1,&bp2, &bp3,&bp4, &bp5}}

	expected := make([]lat.Blueprint,5)
	for i,bp := range replies[4].Next {
		expected[i]=lat.GetBlueprint(*bp)
	}
	return replies, expected
}

func TestGetBlueprintSliceSmall(t *testing.T) {
	replies := make([]*pb.ReadNReply,1)
	replies[0]=&pb.ReadNReply{[]*pb.Blueprint{&bp1}}

	expected := []lat.Blueprint{lat.GetBlueprint(*(*replies[0]).Next[0])}

	result := GetBlueprintSlice(replies)
	for i := range *result {
		if !((*result)[i].Equals(expected[i])) {
			t.Fatalf("GetBlueprint returned at index %d  returned: %v, expected: %v.\n",i, (*result)[i], expected[i])
		}
	}
}

func TestGetBlueprintSlice(t *testing.T) {
	replies, expected := Setup()
	result := GetBlueprintSlice(replies)
	for i := range *result {
		if !((*result)[i].Equals(expected[i])) {
			fmt.Printf("Input 0 is: %v, expected: %v \n",replies[0], expected[0])
			t.Fatalf("GetBlueprint returned at index %d  returned: %v, expected: %v.\n",i, (*result)[i], expected[i])
		}
	}
}
