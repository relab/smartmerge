package rpc

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"testing"
	"time"

	"golang.org/x/net/context"

	pb "github.com/relab/grpc-test/proto"

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

	serverOne := grpc.NewServer()
	serverTwo := grpc.NewServer()

	pb.RegisterRegisterServer(serverOne, &registerServer{})
	pb.RegisterRegisterServer(serverTwo, &registerServer{})

	lisOne, err := net.Listen("tcp", addr1)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	lisTwo, err := net.Listen("tcp", addr2)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	go func() {
		serverOne.Serve(lisOne)
	}()
	go func() {
		serverTwo.Serve(lisTwo)
	}()
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

	// Example: Read configuration, quorum=1, n=1
	myReadConfig, err := mgr.NewConfiguration(ids, 1)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Example: Write configuration, quorum=2, n=2
	myWriteConfig, err := mgr.NewConfiguration(ids, 2)
	if err != nil {
		t.Fatalf("%v", err)
	}

	wreplies, err := myWriteConfig.Write(context.Background(), &pb.State{Value: "42", Timestamp: time.Now().Unix()})
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	fmt.Println("Write replies:", wreplies)

	rreplies, err := myReadConfig.Read(context.Background(), &pb.ReadRequest{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	fmt.Println("Read replies:", rreplies)
}
