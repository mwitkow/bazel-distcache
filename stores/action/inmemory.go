package action

import (
	"github.com/mwitkow/bazel-distcache/common/util"
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"sync"
)

// NewOnDisk constructs *very* naive storage of ActionResults.
// No persistence, no expiration, just a lot of YOLO.
func NewInMemory() Store {
	return &inMemory{values: make(map[string]*remoteexecution.ActionResult)}
}

type inMemory struct {
	mu     sync.RWMutex
	values map[string]*remoteexecution.ActionResult
}

func (s *inMemory) Get(actionDigest *remoteexecution.Digest) (*remoteexecution.ActionResult, error) {
	key := util.ContentDigestToBase64(actionDigest)
	s.mu.RLock()
	val, exists := s.values[key]
	s.mu.RUnlock()
	if !exists {
		return nil, grpc.Errorf(codes.NotFound, "action doesnt exist")
	}
	return val, nil
}

func (s *inMemory) Store(actionDigest *remoteexecution.Digest, actionResult *remoteexecution.ActionResult) error {
	key := util.ContentDigestToBase64(actionDigest)
	s.mu.Lock()
	s.values[key] = actionResult
	s.mu.Unlock()
	return nil
}
