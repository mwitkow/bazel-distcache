package cas

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"
	"github.com/mwitkow/bazel-distcache/stores/action"

	"github.com/mwitkow/bazel-distcache/stores/blob"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// NewLocal builds the CaS gRPC service for local daemon.
func NewLocal() build_remote.CasServiceServer {

	return &local{action.NewInMemory()}
}

type local struct {
	store blob.Store
}

func (s *local) Lookup(ctx context.Context, req *build_remote.CasLookupRequest) (*build_remote.CasLookupReply, error) {
	missing := []*build_remote.ContentDigest{}
	for _, digest := range req.GetDigest() {
		exists, err := s.store.Exists(ctx, digest)
		if err != nil {
			return &build_remote.CasLookupReply{Status: statusFromError(err)}, nil
		}
		if !exists {
			missing = append(missing, digest)
		}
	}
	if len(missing) == 0 {
		return &build_remote.CasLookupReply{Status: statusSuccess}, nil
	} else {
		// it doesn't really matter, even if something is missing we return an error.
		// see https://github.com/bazelbuild/bazel/blob/1575652972d80f224fb3f7398eef3439e4f5a5dd/src/main/java/com/google/devtools/build/lib/remote/GrpcActionCache.java#L245
		return &build_remote.CasLookupReply{
			Status: &build_remote.CasStatus{
				Succeeded: false,
				Error: build_remote.CasStatus_MISSING_DIGEST,
				MissingDigest: missing,
			},
		}, nil
	}

}

func (*local) UploadTreeMetadata(context.Context, *build_remote.CasUploadTreeMetadataRequest) (*build_remote.CasUploadTreeMetadataReply, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "tree processing is not implemented yet")
}


func (*local) DownloadTreeMetadata(context.Context, *build_remote.CasDownloadTreeMetadataRequest) (*build_remote.CasDownloadTreeMetadataReply, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "tree processing is not implemented yet")
}

func (*local) DownloadTree(*build_remote.CasDownloadTreeRequest, build_remote.CasService_DownloadTreeServer) error {
	return  grpc.Errorf(codes.Unimplemented, "tree processing is not implemented yet")
}

func (*local) UploadBlob(build_remote.CasService_UploadBlobServer) error {
	panic("implement me")
}

func (*local) DownloadBlob(*build_remote.CasDownloadBlobRequest, build_remote.CasService_DownloadBlobServer) error {
	panic("implement me")
}

