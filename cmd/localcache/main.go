package main

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"github.com/mwitkow/bazel-distcache/common/sharedflags"
	log "github.com/Sirupsen/logrus"
	"os"
	"google.golang.org/grpc"
	"net"
	"fmt"
	"github.com/mwitkow/bazel-distcache/service/executioncache"
	"google.golang.org/grpc/grpclog"
	"github.com/mwitkow/bazel-distcache/service/cas"
)


var (
	port = sharedflags.Set.Int32("port", 10101, "por tto run the localcache on")
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	if err := sharedflags.Set.Parse(os.Args); err != nil {
		log.Fatalf("failed parsing flags: %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalf("failed listening on 127.0.0.1:%d: %v", *port, err)
	}
	grpclog.SetLogger(log.StandardLogger())
	grpcServer := grpc.NewServer()
	build_remote.RegisterExecutionCacheServiceServer(grpcServer, executioncache.NewLocal())
	build_remote.RegisterCasServiceServer(grpcServer, cas.NewLocal())

	log.Infof("listening for insecure gRPC on: %v", listener.Addr().String())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed staring gRPC server: %v", err)
	}
}