package rpc

import (
	"net"

	"google.golang.org/grpc"
)

type managerOptions struct {
	locationMapper func(ip net.IP) string
	grpcDialOpts   []grpc.DialOption
}

type ManagerOption func(*managerOptions)

func WithLocationMapper(lm func(ip net.IP) string) ManagerOption {
	return func(o *managerOptions) {
		o.locationMapper = lm
	}
}

func WithGrpcDialOptions(opts ...grpc.DialOption) ManagerOption {
	return func(o *managerOptions) {
		o.grpcDialOpts = opts
	}
}
