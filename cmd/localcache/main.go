package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" //registers "/debug/pprof"
	"os"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/mwitkow/bazel-distcache/common/sharedflags"
	"github.com/mwitkow/bazel-distcache/service/actioncache"
	"github.com/mwitkow/bazel-distcache/service/cas"
	"github.com/prometheus/client_golang/prometheus"
	logrus "github.com/sirupsen/logrus"
	_ "golang.org/x/net/trace" // registers /debug/requests
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc"
)

var (
	grpcPort           = sharedflags.Set.Int32("grpc_port", 10101, "grpc (bazel) port to run on")
	httpPort           = sharedflags.Set.Int32("http_port", 10100, "http (debug) port to run on")
	grpcTracingEnabled = sharedflags.Set.Bool("grpc_tracing_enabled", false, "traces whole requests in /debug/request (expensive due to blobs)")
)

func main() {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
	if err := sharedflags.Set.Parse(os.Args); err != nil {
		logrus.Fatalf("failed parsing flags: %v", err)
	}

	grpcListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *grpcPort))
	if err != nil {
		logrus.Fatalf("failed listening on 127.0.0.1:%d: %v", *grpcPort, err)
	}
	httpListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *httpPort))
	if err != nil {
		logrus.Fatalf("failed listening on 127.0.0.1:%d: %v", *httpPort, err)
	}

	logrusEntry := logrus.NewEntry(logrus.StandardLogger())
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)
	grpcServer := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_prometheus.UnaryServerInterceptor,
			grpc_logrus.UnaryServerInterceptor(logrusEntry),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_prometheus.StreamServerInterceptor,
			grpc_logrus.StreamServerInterceptor(logrusEntry),
		),
	)
	grpc.EnableTracing = *grpcTracingEnabled

	casInstance := cas.NewLocal()
	remoteexecution.RegisterActionCacheServer(grpcServer, actioncache.NewLocal())
	remoteexecution.RegisterContentAddressableStorageServer(grpcServer, casInstance)
	bytestream.RegisterByteStreamServer(grpcServer, casInstance)

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
		logrus.Infof("listening for HTTP (debug) on: http://%v", httpListener.Addr().String())
		http.Serve(httpListener, http.DefaultServeMux)
	}()

	logrus.Infof("listening for gRPC (bazel) on: %v", grpcListener.Addr().String())
	if err := grpcServer.Serve(grpcListener); err != nil {
		logrus.Fatalf("failed staring gRPC server: %v", err)
	}
}
