package smclient

import (
	"github.com/golang/glog"
	pb "github.com/relab/smartMerge/proto"
	conf "github.com/relab/smartMerge/confProvider"
)

func (smc *SmClient) get(cp conf.Provider) (rs *pb.State, cnt int) {
	cur := 0
	var rid []uint32
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		cnf := cp.ReadC(smc.Blueps[i], rid)

		read := new(pb.AReadSReply)
		var err error

		for j := 0; cnf != nil; j++ {
			read, err = cnf.AReadS(&pb.Conf{
				This: uint32(smc.Blueps[i].Len()),
				Cur:  uint32(smc.Blueps[cur].Len()),
			})
			cnt++

			if err != nil && j == 0 {
				glog.Errorln("error from OptimizedReadS: ", err)
				// Try again with full configuration.
				cnf = cp.FullC(smc.Blueps[i])
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

		cur = smc.HandleNewCur(cur, read.Reply.GetCur())

		if rs.Compare(read.Reply.GetState()) == 1 {
			rs = read.Reply.GetState()
		}

		if len(smc.Blueps) > i+1 && (read.Reply.GetCur() == nil || !read.Reply.Cur.Abort) {
			rid = pb.Union(rid, read.MachineIDs)
		}

	}

	smc.SetNewCur(cur)
	return
}

func (smc *SmClient) set(cp conf.Provider, rs *pb.State) (cnt int) {
	cur := 0
	var rid []uint32
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		cnf := cp.WriteC(smc.Blueps[i], rid)

		write := new(pb.AWriteSReply)
		var err error

		for j := 0; cnf != nil; j++ {
			write, err = cnf.AWriteS(&pb.WriteS{
				State: rs,
				Conf: &pb.Conf{
					This: uint32(smc.Blueps[i].Len()),
					Cur:  uint32(smc.Blueps[cur].Len()),
				},
			})
			cnt++

			if err != nil && j == 0 {
				glog.Errorln("error from OptimizedWriteS: ", err)
				// Try again with full configuration.
				cnf = cp.FullC(smc.Blueps[i])
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

		cur = smc.HandleNewCur(cur, write.Reply)

		if len(smc.Blueps) > i+1 && (write.Reply == nil || !write.Reply.Abort) {
			rid = pb.Union(rid, write.MachineIDs)
		}

	}

	smc.SetNewCur(cur)
	return cnt
}
