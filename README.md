# smartMerge

This repository contains an implementation of different algorithms for reconfigurable atomic storage.
We implemented our SmartMerge algorithm, Rambo, DynaStore and SpSnStore, which uses the Speculating Snapshot algorithm.

The implementation utilizes our quorum-rpcs framework gorums.
Se github.com/relab/gorums

Howto Run: 
To start a server running the SmartMerge algorithm use:
go run server/server.go -alg=sm


To start a client use 
go run client/client.go -conf server/addrList -alg=sm

