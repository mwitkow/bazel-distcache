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

// benignStatusFor returns a CasStatus that is *benign* to the execution of a stream.
// The download/upload process fails with a RuntimeException, killing bazel:
// https://github.com/bazelbuild/bazel/blob/1575652972d80f224fb3f7398eef3439e4f5a5dd/src/main/java/com/google/devtools/build/lib/remote/GrpcActionCache.java#L295
func benignStatusFor(digest *build_remote.ContentDigest, err error) *build_remote.CasStatus {
	return &build_remote.CasStatus{
		Succeeded:     false,
		MissingDigest: []*build_remote.ContentDigest{digest},
		Error:         build_remote.CasStatus_MISSING_DIGEST,
		ErrorDetail:   err.Error(),
	}
}
