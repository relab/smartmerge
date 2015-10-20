package dynaclient

import (
	"fmt"

	pb "github.com/relab/smartMerge/proto"
)

type OrgDynaClient struct {
	DynaClient
}

func NewOrg(initBlp *pb.Blueprint, mgr *pb.Manager, id uint32) (*OrgDynaClient, error) {
	dc, err := New(initBlp, mgr, id)
	if err != nil {
		return nil, err
	}
	return &OrgDynaClient{*dc}, nil
}

//Atomic read
func (dc *OrgDynaClient) Read() (val []byte, cnt int) {
	val, cnt, err := dc.OrgTraverse(nil, nil)
	if err != nil {
		fmt.Println("OrgTraverse returned error: ", err)
	}
	return val, cnt
}

func (dc *OrgDynaClient) Write(val []byte) int {
	_, cnt, err := dc.OrgTraverse(nil, val)
	if err != nil {
		fmt.Println("OrgTraverse returned error: ", err)
	}
	return cnt
}

func (dc *OrgDynaClient) Reconf(bp *pb.Blueprint) (int, error) {
	_, cnt, err := dc.OrgTraverse(bp, nil)
	return cnt, err
}

func (dc *OrgDynaClient) GetCur() *pb.Blueprint {
	if len(dc.Blueps) == 0 {
		return nil
	}
	return dc.Blueps[0]
}
