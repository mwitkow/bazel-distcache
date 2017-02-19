package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/mwitkow/bazel-distcache/common/sharedflags"
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"github.com/mwitkow/bazel-distcache/service/cas"
	"github.com/mwitkow/bazel-distcache/service/executioncache"
	"github.com/prometheus/client_golang/prometheus"
	_ "golang.org/x/net/trace" // registers /debug/requests
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
	"net/http"
	_ "net/http/pprof" //registers "/debug/pprof"
	"os"
)

var (
	grpcPort           = sharedflags.Set.Int32("grpc_port", 10101, "grpc (bazel) port to run on")
	httpPort           = sharedflags.Set.Int32("http_port", 10100, "http (debug) port to run on")
	grpcTracingEnabled = sharedflags.Set.Bool("grpc_tracing_enabled", false, "traces whole requests in /debug/request (expensive due to blobs)")
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	if err := sharedflags.Set.Parse(os.Args); err != nil {
		log.Fatalf("failed parsing flags: %v", err)
	}

	grpcListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *grpcPort))
	if err != nil {
		log.Fatalf("failed listening on 127.0.0.1:%d: %v", *grpcPort, err)
	}
	httpListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *httpPort))
	if err != nil {
		log.Fatalf("failed listening on 127.0.0.1:%d: %v", *httpPort, err)
	}

	grpclog.SetLogger(log.StandardLogger())
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	grpc.EnableTracing = *grpcTracingEnabled
	build_remote.RegisterExecutionCacheServiceServer(grpcServer, executioncache.NewLocal())
	build_remote.RegisterCasServiceServer(grpcServer, cas.NewLocal())
	grpc_prometheus.Register(grpcServer)

	http.Handle("/metrics", prometheus.UninstrumentedHandler())
	http.Handle("/", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("content-type", "text/plain")
		resp.WriteHeader(http.StatusOK)
		fmt.Fprintf(resp, "Debug interface of localcache\n")
		fmt.Fprintf(resp, "Use command:\n")
		fmt.Fprintf(resp, "\tbazel --host_jvm_args=-Dbazel.DigestFunction=SHA1 --spawn_strategy=remote --remote_cache=localhost:%d build", *grpcPort)
	}))

	go func() {
		log.Infof("listening for HTTP (debug) on: http://%v", httpListener.Addr().String())
		http.Serve(httpListener, http.DefaultServeMux)
	}()

	log.Infof("listening for gRPC (bazel) on: %v", grpcListener.Addr().String())
	if err := grpcServer.Serve(grpcListener); err != nil {
		log.Fatalf("failed staring gRPC server: %v", err)
	}
}
