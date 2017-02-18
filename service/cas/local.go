package cas

import (
	"github.com/mwitkow/bazel-distcache/proto/build/remote"

	"github.com/mwitkow/bazel-distcache/stores/blob"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"io"
	"github.com/mwitkow/bazel-distcache/common/sharedflags"
	log "github.com/Sirupsen/logrus"
)
var (
	chunkSizeBytes = sharedflags.Set.Int("casservice_local_chunk_size_bytes",
		2 * 1024 * 1024,
		"Size of chunk streamed down to bazel clients. Can be max 4MB due to gRPC limits.")
)

// NewLocal builds the CaS gRPC service for local daemon.
func NewLocal() build_remote.CasServiceServer {
	store, err := blob.NewOnDisk()
	if err != nil {
		log.Fatalf("could not initialise CaSService: %v", err)
	}
	return &local{store}
}

type local struct {
	store blob.Store
}

func (s *local) Lookup(ctx context.Context, req *build_remote.CasLookupRequest) (*build_remote.CasLookupReply, error) {
	log.WithField("service", "cas").Infof("Hit Lookup")
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

func (s *local) UploadBlob(stream build_remote.CasService_UploadBlobServer) error {
	log.WithField("service", "cas").Infof("Hit UploadBlob")
	var currentBlob blob.Writer
	for {
		reqFrame, err := stream.Recv()
		if err == io.EOF {
			continue
		} else if err != nil {
			return grpc.Errorf(codes.Unknown, "can't read from stream: %v", err)
		}
		chunk := reqFrame.GetData()
		if chunk == nil {
			return grpc.Errorf(codes.InvalidArgument, "upload blob request must have chunk field")
		}
		if chunk.Offset == 0 && chunk.GetDigest() == nil {
			return grpc.Errorf(codes.InvalidArgument, "upload blob chunk with offset 0 and no digest info")
		}
		if chunk.Offset != 0 && currentBlob == nil {
			return grpc.Errorf(codes.InvalidArgument, "upload blob chunk with non zero offset and no prior blob")
		}
		// Start of first chunk of a blob.
		if chunk.Offset == 0 {
			if currentBlob != nil {
				// Make sure we close the previous blob, and flush stuff etc.
				currentBlob.Close()
			}
			currentBlob, err = s.store.Write(stream.Context(), chunk.GetDigest())
			if err != nil {
				return grpc.Errorf(codes.Internal, "failed opening blob: %v", err)
			}
		}
		wrote, err := currentBlob.Write(chunk.Data)
		if err != nil {
			return grpc.Errorf(codes.Internal, "failed writing blob: %v", err)
		}
		if wrote != len(chunk.Data) {
			return grpc.Errorf(codes.Internal, "bad writer implementation, wrote partially %d of %d", wrote, len(chunk.Data))
		}
	}
	if currentBlob != nil {
		currentBlob.Close()
	}
	stream.SendAndClose(&build_remote.CasUploadBlobReply{Status: statusSuccess})
	return nil
}

func (s *local) DownloadBlob(req *build_remote.CasDownloadBlobRequest, stream build_remote.CasService_DownloadBlobServer) error {
	log.WithField("service", "cas").Infof("Hit DownloadBlob")

	// Base on https://github.com/bazelbuild/bazel/blob/1575652972d80f224fb3f7398eef3439e4f5a5dd/src/main/java/com/google/devtools/build/lib/remote/GrpcActionCache.java#L313
	// It is clear that we *need* to send the chunks down in *exactly* the same order we got them in.
	for _, blobDigest := range req.GetDigest() {
		resp := &build_remote.CasDownloadReply{}
		reader, err := s.store.Read(stream.Context(), blobDigest)
		if err != nil {
			resp.Status = statusFromError(err)
			resp.Status.MissingDigest = append(resp.Status.MissingDigest, blobDigest)
			if err := stream.Send(resp); err != nil {
				return grpc.Errorf(codes.Unknown, "can't write to stream: %v", err)
			}
			continue
		}
		resp.Data = &build_remote.BlobChunk{
			Digest: reader.Digest(), // NOTE: this one contains length!
			Offset: 0,
		}
		// Flush a response frame with just the digest, and continue flushing just the Data contents in subsequent ones.
		// This initial frame *could* hold data as well, but it would make it harder to implement, and I'm lazy.
		// See https://github.com/bazelbuild/bazel/blob/1575652972d80f224fb3f7398eef3439e4f5a5dd/src/main/java/com/google/devtools/build/lib/remote/GrpcActionCache.java#L338
		if err := stream.Send(resp); err != nil {
			return grpc.Errorf(codes.Unknown, "can't write to stream: %v", err)
		}
		offset := int64(0)
		for {
			resp = &build_remote.CasDownloadReply{}
			// TODO(mwitkow): This allocates a lot, try moving it to the top.
			chunkBuffer := make([]byte, *chunkSizeBytes)
			n, err := reader.Read(chunkBuffer)
			if err == io.EOF || n == 0 {
				break
			} else if err != nil {
				resp.Status = statusFromError(err)
				if err := stream.Send(resp); err != nil {
					return grpc.Errorf(codes.Unknown, "can't write to stream: %v", err)
				}
				break
			}
			resp.Data = &build_remote.BlobChunk{
				Offset: offset,
				Data: chunkBuffer[:n],
			}
			offset += int64(n)
			if err := stream.Send(resp); err != nil {
				return grpc.Errorf(codes.Unknown, "can't write to stream: %v", err)
			}
		}
		reader.Close()
	}
	return nil
}

