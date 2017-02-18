package regserver

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	bp "github.com/relab/smartMerge/blueprints"
	pb "github.com/relab/smartMerge/proto"
	grpc "google.golang.org/grpc"
)

// For now, I don't imagine needing to start several servers.
// We therefore do this globally, instead of simply returning the grpcServer.
var grpcServer *grpc.Server
var mu sync.Mutex
var haveServer = false

// Stop stops the grpc server.
func Stop() error {
	mu.Lock()
	defer mu.Unlock()

	if haveServer == false {
		log.Println("Tried to stop grpc-server, but no server was found.")
		return errors.New("No grpc server found.")
	}

	grpcServer.Stop()
	haveServer = false
	return nil
}

// Start a RegServer, as grpc server.
func Start(port int, noabort bool) (*RegServer, error) {
	return StartInConf(port, nil, uint32(0), noabort)
}

// StartInConf starts a RegServer, as grpc server with special initial configuration.
func StartInConf(port int, init *bp.Blueprint, initC uint32, noabort bool) (*RegServer, error) {
	mu.Lock()
	defer mu.Unlock()
	if haveServer == true {
		log.Println("Abort start of grpc server, since old server exists.")
		return nil, errors.New("There already exists an old server.")
	}

	rs := NewRegServerWithCur(init, initC, noabort)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer = grpc.NewServer(opts...)
	pb.RegisterSMandConsRegisterServer(grpcServer, rs)
	go grpcServer.Serve(lis)
	haveServer = true

	return rs, nil
}
