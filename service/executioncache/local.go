package executioncache

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"golang.org/x/net/context"
	"github.com/mwitkow/bazel-distcache/stores/action"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"

	log "github.com/Sirupsen/logrus"

)

// NewLocal builds the ExecutionCache gRPC service for local daemon.
func NewLocal() build_remote.ExecutionCacheServiceServer {
	return &local{action.NewInMemory()}
}

type local struct {
	store action.Store
}

func (s*local) GetCachedResult(ctx context.Context, req *build_remote.ExecutionCacheRequest) (*build_remote.ExecutionCacheReply, error) {
	log.WithField("service", "execcache").Infof("Hit GetCachedResult")
	if req.GetActionDigest() == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "action result must be set")
	}
	actionResult, err := s.store.Get(req.GetActionDigest())
	if err != nil {
		return &build_remote.ExecutionCacheReply{Status: statusFromError(err)}, nil
	}
	return &build_remote.ExecutionCacheReply{Status: statusSuccess, Result: actionResult}, nil
}

func (s*local) SetCachedResult(ctx context.Context, req *build_remote.ExecutionCacheSetRequest) (*build_remote.ExecutionCacheSetReply, error) {
	log.WithField("service", "execcache").Infof("Hit SetCachedResult")
	if req.GetActionDigest() == nil || req.GetResult() == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "action result and its digest must be set")
	}
	err := s.store.Store(req.GetActionDigest(), req.GetResult())
	if err != nil {
		return &build_remote.ExecutionCacheSetReply{Status: statusFromError(err)}, nil
	}
	return &build_remote.ExecutionCacheSetReply{Status: statusSuccess}, nil
}



