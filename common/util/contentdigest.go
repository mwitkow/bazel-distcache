package util

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"encoding/base64"
	"fmt"
)

func ContentDigestToBase64(digest *build_remote.ContentDigest) string {
	return fmt.Sprintf("v%d_%s", digest.Version, base64.RawStdEncoding.EncodeToString(digest.Digest))
}
