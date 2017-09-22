package util

import (
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"strings"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"strconv"
)

const (
	digestFilenameVersion = 1
	blobsResourceField = "blobs/"
)


func ContentDigestToBase64(digest *remoteexecution.Digest) string {
	return fmt.Sprintf("v%d_%s", digestFilenameVersion, digest.Hash)
}



// ResourceToContentDigest translates the bytestream resource name into a Digest object.
//
// See `resource_name` in the documentation of `ContentAddressableStorage`.
//  * {instance_name}/blobs/{hash}/{size}
//  * {instance_name}/uploads/{uuid}/blobs/{hash}/{size}/foo/bar/baz.cc
//  * {instance_name}/blobs/{hash}/{size}
func ResourcePathToContentDigest(resourceName string) (*remoteexecution.Digest, error) {
	var err error
	blobsOffset := strings.Index(resourceName, blobsResourceField)
	if blobsOffset == -1 {
		return nil, status.Errorf(codes.InvalidArgument, "bytestream resource must contain 'blobs/'");
	}

	parts := strings.SplitN(resourceName[blobsOffset:], "/", 4)
	if len(parts) < 3 {
		return nil, status.Errorf(codes.InvalidArgument, "bytestream resource doesn't have enough parts")
	}
	ret := &remoteexecution.Digest{}
	ret.Hash = parts[1]
	ret.SizeBytes, err  = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bytestream size can't be parsed: %v", err)
	}
	return ret, nil
}

func traceFromCtx(ctx context.Context) trace.Trace {
	tr, ok := trace.FromContext(ctx)
	if ok {
		return tr
	}
	// We should never get here. This could leak memory but good for now.
	return trace.New("dummy", "dummy")
}
