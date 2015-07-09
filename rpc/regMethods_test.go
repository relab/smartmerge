package rpc

import (
	"io/ioutil"
	"log"
	"testing"
	"time"
	"fmt"

	"github.com/relab/smartMerge/regserver"
	pb "github.com/relab/smartMerge/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

func init() {
	silentLogger := log.New(ioutil.Discard, "", log.LstdFlags)
	grpclog.SetLogger(silentLogger)
}

func TestRegisterCall(t *testing.T) {
	addr1 := "127.0.0.1:8080"
	addr2 := "127.0.0.1:9090"

	serv1,err1 := regserver.StartTest(8080)
	serv2,err2 := regserver.StartTest(9090)
	defer serv1.Stop()
	defer serv2.Stop()


	if err1 != nil || err2 != nil {
		t.Fatalf("Error creating server")
	}

	time.Sleep(50 * time.Millisecond)

	mgr, err := NewManager(
		[]string{addr1, addr2},
		WithGrpcDialOptions(
			grpc.WithBlock(),
			grpc.WithTimeout(50*time.Millisecond),
		),
	)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// All ids (2 machines)
	ids := mgr.IDs()

	// Example: Write configuration, quorum=2, n=2
	myWriteConfig, err := mgr.NewConfiguration(ids, 2)
	if err != nil {
		t.Fatalf("%v", err)
	}

	s,err := myWriteConfig.ReadS()
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	fmt.Print(s)
	s = pb.State{nil,2}
	err = myWriteConfig.WriteS(&s)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	sr,err := myWriteConfig.ReadS()
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if sr.Timestamp != s.Timestamp {
		t.Errorf("return was %v, expected %v.", sr, s)
	}
}
