package directCombineLattice

import (
	pb "github.com/relab/smartMerge/proto"
)

func (b Blueprint) toMsg() (msg pb.Blueprint) {
	msg.Add = make([]uint32,len(b.Add),0)
	msg.Rem = make([]uint32, len(b.Rem),0)
	for id := range b.Add {
		msg.Add = append(msg.Add, uint32(id))
	}
	for id := range b.Rem {
		msg.Rem = append(msg.Rem, uint32(id))
	}
	return msg
}

func getBlueprint(msg pb.Blueprint) (b Blueprint) {
	b.Add = make( map[ID]bool, len(msg.Add) )
	b.Rem = make( map[ID]bool, len(msg.Rem) )
	for i := range msg.Add {
		b.Add[ID(i)] = true
	}
	for i := range msg.Rem {
		b.Rem[ID(i)] = true
	}
	return b
}
