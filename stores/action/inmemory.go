package action

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"sync"
	"github.com/mwitkow/bazel-distcache/common/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// NewInMemory constructs *very* naive storage of ActionResults.
// No persistence, no expiration, just a lot of YOLO.
func NewInMemory() Store {
	return &inMemory{values: make(map[string]*build_remote.ActionResult)}
}

type inMemory struct {
	mu sync.RWMutex
	values map[string]*build_remote.ActionResult
}

func (s *inMemory) Get(actionDigest *build_remote.ContentDigest) (*build_remote.ActionResult, error) {
	key := util.ContentDigestToBase64(actionDigest)
	s.mu.RLock()
	val, exists := s.values[key]
	s.mu.RUnlock()
	if !exists {
		return nil, grpc.Errorf(codes.NotFound, "action doesnt exist")
	}
	return val, nil
}

func (s *inMemory) Store(actionDigest *build_remote.ContentDigest, actionResult *build_remote.ActionResult) error {
	key := util.ContentDigestToBase64(actionDigest)
	s.mu.Lock()
	s.values[key] = actionResult
	s.mu.Unlock()
	return nil
}

