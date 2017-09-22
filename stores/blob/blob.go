package blob

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"golang.org/x/net/context"
	"io"
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

// Store is a general interface for storing ActionResults.
// ActionResults are descriptions of bazel build actions (steps), with a set of outputs (held in content store),
// the exit code and stdout/stderr results (also held in content store).
// By itself the objects are small (kilobytes at best), and act as pointers to content store.
type Store interface {
	// Exists returns whether the given blob exists for its digest.
	// If errors occur, they are reported as errors, and should fail builds.
	Exists(ctx context.Context, blobDigest *remoteexecution.Digest) (bool, error)
	// Read returns a Reader for the digest.
	// Must return grpc.NotFound error if no blob exists. Other errors will cause failure of builds.
	Read(ctx context.Context, blobDigest *remoteexecution.Digest) (Reader, error)
	// Write returns a BlogWriter for the digest.
	Write(ctx context.Context, blobDigest *remoteexecution.Digest) (Writer, error)
}

type digestGetter interface {
	// Digest returns the content digest of the blob served.
	Digest() *remoteexecution.Digest
}

// Reader is an interface for accessing blobs in store.
// Each Read is guaranteed to fill the buffer before returning, unless there are fewer items remaining or an error occured.
// Users *must* call Close() when they're done reading.
type Reader interface {
	io.ReadCloser
	digestGetter
}

// Writer is an interface for writing blob contents into the store.
// Each Write is guranteed to write the whole buffer, unless an error occurs.
// Users *must* call Close() when they're done writing.
type Writer interface {
	io.WriteCloser
	digestGetter
}
