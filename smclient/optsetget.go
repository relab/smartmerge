package smclient

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
)

const Retry = 1

func (smc *SmOptClient) get() (rs *pb.State, cnt int) {
	cur := 0
	var rid []uint32
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		cnf := smc.getReadC(i, rid)

		read := new(pb.AReadSReply)
		var err error

		for j := 0; cnf != nil; j++ {
			read, err = cnf.AReadS(&pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[cur].Len())})
			cnt++

			if err != nil && j == 0 {
				glog.Errorln("error from OptimizedReadS: ", err)
				// Try again with full configuration.
				cnf = smc.getFullC(i)
			}

			if err != nil && j == Retry {
				glog.Errorf("error %v from ReadS after %d retries.\n", err, Retry)
				return nil, 0
			}

			if err == nil {
				break
			}
		}

		if glog.V(6) {
			glog.Infoln("AReadS returned with replies from ", read.MachineIDs)
		}

		cur = smc.handleNewCur(cur, read.Reply.GetCur(), false)

		if rs.Compare(read.Reply.GetState()) == 1 {
			rs = read.Reply.GetState()
		}

		if len(smc.Blueps) > i+1 && (read.Reply.GetCur() == nil || !read.Reply.Cur.Abort) {
			rid = pb.Union(rid, read.MachineIDs)
		}

	}

	smc.setNewCur(cur)
	return
}

func (smc *SmOptClient) set(rs *pb.State) (cnt int) {
	cur := 0
	var rid []uint32
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		cnf := smc.getWriteC(i, rid)

		write := new(pb.AWriteSReply)
		var err error

		for j := 0; cnf != nil; j++ {
			write, err = cnf.AWriteS(&pb.WriteS{rs, &pb.Conf{uint32(smc.Blueps[i].Len()), uint32(smc.Blueps[cur].Len())}})
			cnt++

			if err != nil && j == 0 {
				glog.Errorln("error from OptimizedWriteS: ", err)
				// Try again with full configuration.
				cnf = smc.getFullC(i)
			}

			if err != nil && j == Retry {
				glog.Errorf("error %v from WriteS after %d retries. \n", err, Retry)
				return 0
			}

			if err == nil {
				break
			}
		}

		if glog.V(6) {
			glog.Infoln("AWriteS returned, with replies from ", write.MachineIDs)
		}

		cur = smc.handleNewCur(cur, write.Reply, false)

		if len(smc.Blueps) > i+1 && (write.Reply == nil || !write.Reply.Abort) {
			rid = pb.Union(rid, write.MachineIDs)
		}

	}

	smc.setNewCur(cur)
	return cnt
}
