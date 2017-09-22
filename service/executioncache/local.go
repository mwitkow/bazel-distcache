package executioncache

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"github.com/mwitkow/bazel-distcache/stores/action"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	log "github.com/Sirupsen/logrus"
)

// NewLocal builds the ExecutionCache gRPC service for local daemon.
//
// NOTE: This is *deprecated* in favour of the new implementaiton in actioncache.
func NewLocal() build_remote.ExecutionCacheServiceServer {
	store, err := action.NewOnDisk()
	if err != nil {
		log.Fatalf("could not initialise ExecutionCacheService: %v", err)
	}
	return &local{store}
}

type local struct {
	store action.Store
}

func (s *local) GetCachedResult(ctx context.Context, req *build_remote.ExecutionCacheRequest) (*build_remote.ExecutionCacheReply, error) {
	logger := log.WithField("service", "execcache").WithField("method", "GetCachedResult")
	if req.GetActionDigest() == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "action result must be set")
	}
	actionResult, err := s.store.Get(req.GetActionDigest())
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			logger.Infof("missed")
		} else {
			logger.Warningf("missed: %v", err)
		}
		return &build_remote.ExecutionCacheReply{Status: statusFromError(err)}, nil
	}
	logger.Infof("hit")
	return &build_remote.ExecutionCacheReply{Status: statusSuccess, Result: actionResult}, nil
}

func (s *local) SetCachedResult(ctx context.Context, req *build_remote.ExecutionCacheSetRequest) (*build_remote.ExecutionCacheSetReply, error) {
	logger := log.WithField("service", "execcache").WithField("method", "SetCachedResult")
	if req.GetActionDigest() == nil || req.GetResult() == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "action result and its digest must be set")
	}
	err := s.store.Store(req.GetActionDigest(), req.GetResult())
	if err != nil {
		logger.Warningf("failed: %v", err)
		return &build_remote.ExecutionCacheSetReply{Status: statusFromError(err)}, nil
	}
	logger.Infof("stored")
	return &build_remote.ExecutionCacheSetReply{Status: statusSuccess}, nil
}
