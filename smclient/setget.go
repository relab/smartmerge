package smclient

import (
	"golang.org/x/net/context"

	"github.com/golang/glog"

	bp "github.com/relab/smartMerge/blueprints"
	conf "github.com/relab/smartMerge/confProvider"
	pb "github.com/relab/smartMerge/proto"
)

func (smc *SmClient) get(cp conf.Provider) (rs *pb.State, cnt int) {
	cur := 0

	// rid is used to store ids of nodes, that have already replied.
	// These nodes will be omitted, in calls to additional configurations, if
	// the norecontact configuration provider is used.
	var rid []uint32
	for i := 0; i < len(smc.Blueps); i++ {
		//cnt++
		if i < cur {
			continue
		}
		if i > 0 && i == cur {
			// Asynchronously notify the servers, that a new configuration was installed.
			go smc.SetCur(cp, smc.Blueps[cur])
		}
		smc.checkrid(i, rid, cp)

		cnf := cp.ReadC(smc.Blueps[i], rid)
		glog.Infof("Blueprint %v")
		if cnf == nil {
			cnt++
		}

		read := new(pb.ReadReply_)
		var err error

		for j := 0; cnf != nil; j++ {
			read, err = cnf.Read(context.Background(), &pb.Conf{
				This: uint32(smc.Blueps[i].Hash()),
				Cur:  uint32(smc.Blueps[cur].Hash()),
			})
			cnt++

			if err != nil && j == 0 {
				glog.Errorln("error from OptimizedRead: ", err)
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
			glog.Infoln("Read returned with replies from ", read.NodeIDs)
		}

		// Update list of blueprints.
		cur = smc.HandleNewCur(cur, read.GetCur())

		if rs.Compare(read.GetState()) == 1 {
			rs = read.GetState()
		}

		if len(smc.Blueps) > i+1 && (read.GetCur() == nil || !read.Cur.Abort) {
			rid = bp.Union(rid, read.NodeIDs)
		}

	}

	smc.SetNewCur(cur)
	return
}

func (smc *SmClient) set(cp conf.Provider, rs *pb.State) (cnt int) {
	cur := 0
	var rid []uint32
	for i := 0; i < len(smc.Blueps); i++ {
		//cnt++
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

		write := new(pb.WriteReply)
		var err error

		for j := 0; cnf != nil; j++ {
			write, err = cnf.Write(context.Background(), &pb.WriteS{
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
			glog.Infoln("Write returned, with replies from ", write.NodeIDs)
		}

		cur = smc.HandleNewCur(cur, write.ConfReply)

		if len(smc.Blueps) > i+1 && (write.ConfReply == nil || !write.Abort) {
			rid = bp.Union(rid, write.NodeIDs)
		}

	}

	smc.SetNewCur(cur)
	return cnt
}

// checkrid checks whether one of the nodes that have already replies (stored in rids)
// has been removed from the new configuration/blueprint. In this case, we remove the
// node from rids.
// This ensures, that if a node is removed and then readded, we do recontact that node.
// Thus, a node can be readded, even if it lost some in-memory state.
// Further, when performing benchmarks this ensures that removing and readding the same node
// causes a similar overhead to removing one, and adding a new node.
func (smc *SmClient) checkrid(new int, rids []uint32, cp conf.Provider) []uint32 {
	if new == 0 {
		return nil
	}

	remove := make([]bool, len(rids))
	for k, rid := range rids {
	for_old:
		for _, n := range smc.Blueps[new-1].Nodes {
			if n.Id == rid {
				for _, nn := range smc.Blueps[new].Nodes {
					if nn.Id == rid {
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
	nrid := make([]uint32, 0, len(rids))
	for k, id := range rids {
		if !remove[k] {
			nrid = append(nrid, id)
		}
	}
	return nrid
}
