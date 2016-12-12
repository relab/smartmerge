# The Case for Reconfiguration without Consensus

This repository contains an implementation of different algorithms for reconfigurable atomic storage.
We implemented our SmartMerge algorithm, Rambo, DynaStore and SpSnStore, which uses the Speculating Snapshot algorithm.

The implementation utilizes our quorum-rpcs framework [gorums](http://github.com/relab/gorums).


## Howto Run: 
Clone the repository into your [GOPATH](http://golang.org/doc/install).
```
mkdir $GOPATH/src/github.com/relab
cd $GOPATH/src/github.com/relab
git clone git@github.com:relab/smartMerge.git
```

To get all dependencies use 
```
cd $GOPATH/src/github.com/relab/smartMerge
go get ./...
```

To start a server running the SmartMerge algorithm use:
```
go run server/server.go -alg=sm -port 10011
go run server/server.go -alg=sm -port 10012
go run server/server.go -alg=sm -port 10013
go run server/server.go -alg=sm -port 10014
```
Use `go run server/server.go -help` to display possible options, e.g. different algorithms.

To start an interactive client use 
```
cd $GOPATH/src/github.com/relab/smartMerge/client
go build
./client -conf addrList -alg=sm -initsize=3  //Starts a client with initial configuration containing 3 servers.
```

Use `./client -help`to deplay all possible arguments/options.


