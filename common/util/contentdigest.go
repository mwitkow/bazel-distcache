package util

import (
	"encoding/base64"
	"fmt"
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
)

func ContentDigestToBase64(digest *build_remote.ContentDigest) string {
	return fmt.Sprintf("v%d_%s", digest.Version, base64.RawURLEncoding.EncodeToString(digest.Digest))
}

func traceFromCtx(ctx context.Context) trace.Trace {
	tr, ok := trace.FromContext(ctx)
	if ok {
		return tr
	}
	// We should never get here. This could leak memory but good for now.
	return trace.New("dummy", "dummy")
}
