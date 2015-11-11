package smclient

import (
	"github.com/golang/glog"

	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) setNewCur(cur int) {
	if cur>= len(smc.Blueps) {
		glog.Fatalln("Index for new cur out of bound.")
	}
	
	if cur == 0 {
		return
	}
	
	smc.curRead = smc.getReadC(cur, nil)
	
	smc.curWrite = smc.getWriteC(cur, nil)
	
	smc.Confs[0] = smc.getFullC(cur)
	
	smc.Blueps = smc.Blueps[cur:]	
}

func (smc *SmClient) chooseQ(ids []uint32, q int) (quorum []uint32) {
	if q > len(ids) {
		glog.Fatalf("Trying to choose %d nodes, out of %d\n", q, len(ids))
	}
	
	quorum = make([]uint32,q)
	start := int(smc.ID)%len(ids)
	if start+q <= len(ids) {
		copy(quorum, ids[start:])
		return quorum
	}  
	copy(quorum, ids[start:])
	copy(quorum[len(ids)-start:],ids)
	return quorum
}

func (smc *SmClient) getReadC(i int, rids []uint32) (*pb.Configuration) {
	if i == 0 && len(rids) == 0 {
		return smc.curRead
	}
	
	cids := smc.Blueps[i].Ids()
	rq := len(cids) - smc.Blueps[i].Quorum() + 1 //read quorum
	newcids := pb.Difference(cids, rids)
	
	if len(cids) - len(newcids) >= rq {
		//We already have enough replies.
		return nil
	}
	
	// I already have y := len(cids) - len(newcids) many replies.
	// I still need rq - y many.
	newcids = smc.chooseQ(newcids, rq - len(cids) + len(newcids))
	
	// With quorum size 1, a read quorum contains all processes.
	cnf, err := smc.mgr.NewConfiguration(newcids, 1, TryTimeout)
	if err != nil {
		glog.Fatalln("could not get read config")
	}
	
	return cnf
}

func (smc *SmClient) getWriteC(i int, rids []uint32) (*pb.Configuration) {
	if i == 0 && len(rids) == 0 {
		return smc.curWrite
	}
	
	cids := smc.Blueps[i].Ids()
	q := smc.Blueps[i].Quorum()
	newcids := pb.Difference(cids, rids)
	
	if len(cids) - len(newcids) >= q {
		//We already have enough replies.
		return nil
	}
	
	// I already have y := len(cids) - len(newcids) many replies.
	// I still need q - y many.
	newcids = smc.chooseQ(newcids, q - len(cids) + len(newcids))
	cnf, err := smc.mgr.NewConfiguration(newcids, len(newcids), TryTimeout)
	if err != nil {
		glog.Fatalln("could not get read config")
	}
	
	return cnf
}

func (smc *SmClient) getFullC(i int) (*pb.Configuration) {
	cids := smc.Blueps[i].Ids()
	q := smc.Blueps[i].Quorum()
	
	cnf, err := smc.mgr.NewConfiguration(cids, q, ConfTimeout)
	if err != nil {
		glog.Fatalln("could not get config")
	}
	
	return cnf

}