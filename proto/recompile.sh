#!/bin/sh

#go install github.com/relab/protobuf/...
#protoc --go_out=plugins=grpc+gorums:. dc-smartMerge.proto

protoc --gogofast_out=plugins=grpc+gorums:. dc-smartMerge.proto

from proto folder compile with:
protoc -I=../../../../../:. --gorums_out=plugins=grpc+gorums:. *.proto
