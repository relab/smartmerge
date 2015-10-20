package dynaclient

import (
	//"errors"
	"fmt"
	"time"

	pb "github.com/relab/smartMerge/proto"
)

func (dc *DynaClient) OrgTraverse(prop *pb.Blueprint, val []byte) ([]byte, int, error) {
	cnt := 0
	rst := new(pb.State)
	for i := 0; i < len(dc.Confs); i++ {
		if prop != nil && !prop.Equals(dc.Blueps[i]) {
	
			getOne, err := dc.Confs[i].GetOneN(&pb.GetOne{uint32(dc.Blueps[i].Len()), prop})
			//fmt.Println("invoke getone")
			cnt++
			isnew := dc.abortonNewCur(getOne.Reply.GetCur())
			if isnew {
				prop = prop.Merge(getOne.Reply.GetCur())
				i = -1
				continue
			}
			
			if err != nil {
				fmt.Println("Error from GetOneN")
				return nil, 0, err
			}
			
			curprop := getOne.Reply.GetNext()

			writeN, err := dc.Confs[i].DWriteNSet(&pb.DWriteN{uint32(dc.Blueps[i].Len()),[]*pb.Blueprint{curprop}})
			//fmt.Println("invoke writeN")
			cnt++
			isnew = dc.abortonNewCur(writeN.Reply.GetCur())
			if isnew {
				prop = prop.Merge(writeN.Reply.GetCur())
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteNSet")
				return nil, 0, err
			}
		}

		//ReadInView:
		read, err := dc.Confs[i].DReadS(&pb.DRead{uint32(dc.Blueps[i].Len()), nil})
		//fmt.Println("invoke readS")
		cnt++
		restart := dc.abortonNewCur(read.Reply.GetCur())
		if restart {
			prop = prop.Merge(read.Reply.GetCur())
			i = -1
			continue
		}
		if err != nil {
			fmt.Println("Error from DReadS")
			return nil, 0, err
		}

		//Using DWriteS instead of readN is a shortcut, but works just fine.
		readN, err := dc.Confs[i].DWriteS(&pb.AdvWriteS{nil, uint32(dc.Blueps[i].Len())})
		//fmt.Println("invoke readN")
		cnt++
		restart = dc.abortonNewCur(readN.Reply.GetCur())
		if restart {
			prop = prop.Merge(read.Reply.GetCur())
			i = -1
			continue
		}
		if err != nil {
			fmt.Println("Error from DReadN")
			return nil, 0, err
		}

		next := read.Reply.GetNext()
		prop = dc.handleNext(i, next, prop)
		if rst.Compare(read.Reply.GetState()) == 1 {
			rst = read.Reply.GetState()
		}

		if len(next) == 0 {
			//WriteInView
			wst := dc.WriteValue(val, rst)
			write, err := dc.Confs[i].DWriteS(&pb.AdvWriteS{wst, uint32(dc.Blueps[i].Len())})
			//fmt.Println("invoke writeS")
			cnt++
			restart = dc.abortonNewCur(write.Reply.GetCur())
			if restart {
				prop = prop.Merge(write.Reply.GetCur())
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteS")
				return nil, 0, err
			}

			//Using DWriteS here is a shortcut, but I think it works just fine.
			readN, err = dc.Confs[i].DWriteS(&pb.AdvWriteS{nil, uint32(dc.Blueps[i].Len())})
			//fmt.Println("invoke readN")
			cnt++
			restart = dc.abortonNewCur(readN.Reply.GetCur())
			if restart {
				prop = prop.Merge(readN.Reply.GetCur())
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteS")
				return nil, 0, err
			}

			next = readN.Reply.GetNext()
			prop = dc.handleNext(i, next, prop)

		}

		if len(next) > 0 {
			writeN, err := dc.Confs[i].DWriteNSet(&pb.DWriteN{uint32(dc.Blueps[i].Len()),next})
			//fmt.Println("invoke writeN")
			cnt++
			restart = dc.abortonNewCur(writeN.Reply.GetCur())
			if restart {
				prop = prop.Merge(writeN.Reply.GetCur())
				i = -1
				continue
			}

			if err != nil {
				fmt.Println("Error from DWriteNSet")
				return nil, 0, err
			}
			continue
		}
	}

	
	if i := len(dc.Confs) -1; i > 0  {
		dc.Confs[i].DSetCur(&pb.NewCur{dc.Blueps[i], uint32(dc.Blueps[i].Len())})
		//fmt.Println("setcur")
		cnt++
	
		dc.Blueps = dc.Blueps[i:]
		dc.Confs = dc.Confs[i:]
	}

	if val == nil {
		return rst.Value, cnt, nil
	}
	return nil, cnt, nil
}

func (dc *DynaClient) abortonNewCur(newCur *pb.Blueprint) bool {
	switch newCur.Compare(dc.Blueps[0]) {
	case 1: 
		return false
	case -1:
		cnf, err := dc.mgr.NewConfiguration(newCur.Add, majQuorum(newCur), 2* time.Second)
		if err != nil {
			panic("could not get new config")
		}
		dc.Blueps = []*pb.Blueprint{newCur}
		dc.Confs = []*pb.Configuration{cnf}
		return true
	case 0:
		panic("Old and new current blueprint are not comparable.")
	}
	return false
}