package smclient

import (
	"github.com/golang/glog"
	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) get(cp conf.Provider) (rs *pb.State, cnt int) {
	cur := 0
	var rid []int
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		if i > 0 && i == cur {
			go smc.SetCur(cp, smc.Blueps[cur])
		}
		smc.checkrid(i, rid, cp)

		cnf := cp.ReadC(smc.Blueps[i], rid)
		if cnf == nil {
			cnt++
		}

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
	var rid []int
	for i := 0; i < len(smc.Blueps); i++ {
		if i < cur {
			continue
		}

		if i > 0 && i == cur {
			go smc.SetCur(cp, smc.Blueps[cur])
		}
		smc.checkrid(i, rid, cp)

		cnf := cp.WriteC(smc.Blueps[i], rid)
		if cnf == nil {
			cnt++
		}

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

func (smc *SmClient) checkrid(new int, rid []int, cp conf.Provider) []int {
	if new == 0 {
		return nil
	}

	gids := cp.GIDs(rid)
	if gids == nil {
		return rid
	}

	remove := make([]bool, len(rid))
	for k, gid := range gids {
	for_old:
		for _, n := range smc.Blueps[new-1].Nodes {
			if n.Id == gid {
				for _, nn := range smc.Blueps[new].Nodes {
					if nn.Id == gid {
						if n.Version < nn.Version {
							remove[k] = true
							//remove
						}
						break for_old
					}
				}
			}
		}
	}
	nrid := make([]int, 0, len(rid))
	for k, id := range rid {
		if !remove[k] {
			nrid = append(nrid, id)
		}
	}
	return nrid
}
