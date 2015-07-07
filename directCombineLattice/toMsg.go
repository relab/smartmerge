package directCombineLattice

import (
	pb "github.com/relab/smartMerge/proto"
)

func (b Blueprint) ToMsg() (msg pb.Blueprint) {
	msg.Add = make([]uint32, 0, len(b.Add))
	msg.Rem = make([]uint32, 0, len(b.Rem))
	for id := range b.Add {
		msg.Add = append(msg.Add, uint32(id))
	}
	for id := range b.Rem {
		msg.Rem = append(msg.Rem, uint32(id))
	}
	return msg
}

func GetBlueprint(msg pb.Blueprint) (b Blueprint) {
	b.Add = make(map[ID]bool, len(msg.Add))
	b.Rem = make(map[ID]bool, len(msg.Rem))
	for _, i := range msg.Add {
		b.Add[ID(i)] = true
	}
	for _, i := range msg.Rem {
		b.Rem[ID(i)] = true
	}
	return b
}
