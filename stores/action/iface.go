package action

import (
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)


// Store is a general interface for caching ActionResults.
// ActionResults are descriptions of bazel build actions (steps), with a set of outputs (held in content store),
// the exit code and stdout/stderr results (also held in content store).
// By itself the objects are small (kilobytes at best), and act as pointers to content store.
type Store interface {
	// Get returns an ActionResult by its digest.
	// Must return grpc.NotFound error if no action exists. Other errors will cause failure of builds.
	Get(actionDigest *remoteexecution.Digest) (*remoteexecution.ActionResult, error)

	// Store returns an ActionResult by its digest.
	Store(actionDigest *remoteexecution.Digest, actionResult *remoteexecution.ActionResult) error
}
