Howto compile the proto file:

cd $GOPATH/src/github.com/golang/protobuf/protoc-gen-go
protoc -I ../../../relab/smartMerge/proto ../../../relab/smartMerge/proto/dc-smartMerge.proto --go_out=plugins=grpc:../../../relab/smartMerge/proto/
cd -
