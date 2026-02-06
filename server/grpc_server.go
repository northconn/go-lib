package adapters

import "google.golang.org/grpc"

func NewGRPCServer() *grpc.Server {
	return grpc.NewServer()
}
