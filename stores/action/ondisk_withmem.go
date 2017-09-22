package action

import (
	"github.com/mwitkow/bazel-distcache/common/sharedflags"
	"github.com/mwitkow/bazel-distcache/common/util"
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"sync"
	"fmt"
	"io/ioutil"
	"path"
	"os"
	"github.com/golang/protobuf/proto"
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

var (
	diskPath = sharedflags.Set.String("actionstore_ondisk_path", "/tmp/localcache-actionstore", "Path for the ondisk blob store directory.")
)

// NewOnDisk constructs *very* naive storage of Action that is stored in a directory from flags.
// It is backed by on-disk proto messages.
func NewOnDisk() (Store, error) {
	s := &onDisk{values: make(map[string]*remoteexecution.ActionResult), basePath: *diskPath}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

type onDisk struct {
	mu       sync.RWMutex
	basePath string

	values map[string]*remoteexecution.ActionResult
}

func (s *onDisk) init() error {
	files, err := ioutil.ReadDir(s.basePath)
	if err != nil {
		return fmt.Errorf("ondisk actionstore initialization error: %v", err)
	}
	s.mu.Lock()
	for _, f := range files {
		action, err := s.readActionFromDisk(f.Name())
		if err != nil {
			return err
		}
		s.values[f.Name()] = action
	}
	s.mu.Unlock()
	return nil
}

func (s *onDisk) Get(actionDigest *remoteexecution.Digest) (*remoteexecution.ActionResult, error) {
	key := util.ContentDigestToBase64(actionDigest)
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.values[key]
	if exists {
		return val, nil
	}
	ret, err := s.readActionFromDisk(key)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *onDisk) readActionFromDisk(key string) (*remoteexecution.ActionResult, error) {
	content, err := ioutil.ReadFile(path.Join(s.basePath, key))
	if os.IsNotExist(err) {
		return nil, grpc.Errorf(codes.NotFound, "action doesnt exist")
	} else if err != nil {
		return nil, grpc.Errorf(codes.Internal, "ondisk actionstore can't read file %v: %v", key, err)
	}
	res := &remoteexecution.ActionResult{}
	if err := proto.Unmarshal(content, res); err != nil {
		return nil, grpc.Errorf(codes.Internal, "action is unparsable %v: %v", key, err)
	}
	return res, nil
}

func (s *onDisk) Store(actionDigest *remoteexecution.Digest, actionResult *remoteexecution.ActionResult) error {
	key := util.ContentDigestToBase64(actionDigest)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.storeActionToDisk(key, actionResult); err != nil {
		return err
	}
	s.values[key] = actionResult
	return nil
}

func (s *onDisk) storeActionToDisk(key string, actionResult *remoteexecution.ActionResult) error {
	bytes, err := proto.Marshal(actionResult)
	if err != nil {
		return grpc.Errorf(codes.Internal, "action is unmarshable %v: %v", key, err)
	}
	if err := ioutil.WriteFile(path.Join(s.basePath, key), bytes, 0666); err != nil {
		return grpc.Errorf(codes.Internal, "ondisk actionstore can't write file %v: %v", key, err)
	}
	return nil
}
