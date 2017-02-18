package blob

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"sync"
	"github.com/mwitkow/bazel-distcache/common/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"github.com/mwitkow/bazel-distcache/common/sharedflags"
	"os"
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"path"
	"io"
)

const (
	sizeNoExist = -1
)

var (
	diskPath = sharedflags.Set.String("blobstore_ondisk_path", "/tmp/localcache-blobstore", "Path for the ondisk blob store directory.")
)

// NewOnDisk constructs *very* naive storage of Blobs that is stored in a directory from flags.
// No persistence, no expiration, just a lot of YOLO.
func NewOnDisk() (Store, error) {
	s := &onDisk{sizeCache: make(map[string]int64), basePath: *diskPath}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

type onDisk struct {
	mu        sync.RWMutex
	basePath  string
	sizeCache map[string]int64
}

func (s *onDisk) init() error {
	files, err := ioutil.ReadDir(s.basePath)
	if err != nil {
		return fmt.Errorf("ondisk blobstore initialization error: %v", err)
	}
	for _, f := range files {
		s.cacheSize(f.Name())
	}
	return nil
}

func (s *onDisk) getSize(blobKey string) int64 {
	s.mu.RLock()
	value, exists := s.sizeCache[blobKey]
	s.mu.RUnlock()
	if !exists {
		return sizeNoExist
	}
	return value
}

func (s *onDisk) cacheSize(blobKey string, size int64) {
	s.mu.Lock()
	s.sizeCache[blobKey] = size
	s.mu.Unlock()
}

func (s *onDisk) Exists(ctx context.Context, blobDigest *build_remote.ContentDigest) (bool, error) {
	key := util.ContentDigestToBase64(blobDigest)
	return s.getSize(key) != sizeNoExist, nil
}

func (s *onDisk) Read(ctx context.Context, blobDigest *build_remote.ContentDigest) (BlobReader, error) {
	key := util.ContentDigestToBase64(blobDigest)
	fileName := path.Join(s.basePath, key)
	size := s.getSize(key)
	if size == sizeNoExist {
		return nil, grpc.Errorf(codes.NotFound, "blob for contentdigest doesn't exist")
	}
	file, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, grpc.Errorf(codes.NotFound, "blob for contentdigest doesn't exist on disk")
		}
		return nil, grpc.Errorf(codes.Internal, "ondisk blobstore can't open file: %v", err)
	}
	// Make sure we expose the size of the blob stored, since we're reusing the blobDigest object.
	blobDigest.SizeBytes = size
	return &blobFile{digest: blobDigest, file: file}, nil
}

func (s *onDisk) Write(ctx context.Context, blobDigest *build_remote.ContentDigest) (BlobWriter, error) {
	key := util.ContentDigestToBase64(blobDigest)
	fileName := path.Join(s.basePath, key)
	file, err := os.Create(fileName)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "ondisk blobstore can't create file: %v", err)
	}
	s.cacheSize(key, blobDigest.SizeBytes)
	return &blobFile{digest: blobDigest, file: file}, nil
}

// blobFile is a general implementation of both BlobWriter and BlobReader, and which one it is
// depends on the particular mode of the file open underneath
type blobFile struct {
	digest *build_remote.ContentDigest
	file *os.File
}

func (b *blobFile) Read(p []byte) (n int, err error) {
	return io.ReadFull(b.file, p)
}

func (b *blobFile) Write(p []byte) (n int, err error) {
	return b.file.Write(p)
}

func (b *blobFile) Close() error {
	return b.file.Close()
}

func (b *blobFile) Digest() *build_remote.ContentDigest {
	return b.digest
}
