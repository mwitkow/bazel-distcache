package cas

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	statusSuccess = &build_remote.CasStatus{Succeeded: true}
)

func statusFromError(err error) *build_remote.CasStatus {
	status := &build_remote.CasStatus{Succeeded: false}
	// non-gRPC error
	if grpc.Code(err) == codes.Unknown {
		status.Error = build_remote.CasStatus_UNKNOWN
		status.ErrorDetail = err.Error()
		return status
	}
	status.ErrorDetail = grpc.ErrorDesc(err)
	switch grpc.Code(err) {
	case codes.NotFound:
		status.Error = build_remote.CasStatus_MISSING_DIGEST
	default:
		status.Error = build_remote.CasStatus_UNKNOWN
	}
	return status

}

// remappedStatusOrError attempts to move the error to a benign form of CasStatus.
// The bazel implementation immediately the whole execution on any gRPC errors.
func remappedStatusOrError(err error) (*build_remote.CasStatus, error) {
	if grpc.Code(err) == codes.NotFound {
		return statusFromError(err), nil
	}
	return nil, err
}
