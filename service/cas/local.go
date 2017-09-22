package cas

import (
	"io"

	"github.com/mwitkow/bazel-distcache/common/sharedflags"
	"github.com/mwitkow/bazel-distcache/stores/blob"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"

	"io/ioutil"

	"github.com/mwitkow/bazel-distcache/common/util"
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/status"
)

var (
	chunkSizeBytes = sharedflags.Set.Int("casservice_local_chunk_size_bytes",
		2*1024*1024,
		"Size of chunk streamed down to bazel clients. Can be max 4MB due to gRPC limits.")
)

// ConcreteCasServer is a combined implementation of the ByteStreamServer and the ContentAddressableStorageServer.
type ConcreteCaSServer interface {
	remoteexecution.ContentAddressableStorageServer
	bytestream.ByteStreamServer
}

// NewLocal builds the CaS gRPC service for local daemon.
func NewLocal() ConcreteCaSServer {
	store, err := blob.NewOnDisk()
	if err != nil {
		log.Fatalf("could not initialise CaSService: %v", err)
	}
	return &local{store}
}

// local implements both the ContentAddressableStorageService and the BlobStreamService
type local struct {
	store blob.Store
}

func (l *local) FindMissingBlobs(ctx context.Context, req *remoteexecution.FindMissingBlobsRequest) (*remoteexecution.FindMissingBlobsResponse, error) {
	// TODO(mwitkow): Handle instance name of the request resourceName
	resp := &remoteexecution.FindMissingBlobsResponse{}
	for _, blobDigest := range req.BlobDigests {
		exists, err := l.store.Exists(ctx, blobDigest)
		if err != nil {
			return nil, err
		}
		if !exists {
			resp.MissingBlobDigests = append(resp.MissingBlobDigests, blobDigest)
		}
	}
	return resp, nil
}

func (l *local) BatchUpdateBlobs(context.Context, *remoteexecution.BatchUpdateBlobsRequest) (*remoteexecution.BatchUpdateBlobsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "BatchUpdateBlobs is not used in bazel 0.5.3, ignore for now.")
}

func (l *local) GetTree(context.Context, *remoteexecution.GetTreeRequest) (*remoteexecution.GetTreeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetTree is deprecated and unused in bazel >= 0.5.3.")
}

func (l *local) Read(req *bytestream.ReadRequest, readStream bytestream.ByteStream_ReadServer) error {
	// TODO(mwitkow): Handle instance name of the request resourceName
	blobDigest, err := util.ResourcePathToContentDigest(req.ResourceName)
	if err != nil {
		return err
	}
	blobReader, err := l.store.Read(readStream.Context(), blobDigest)
	if err != nil {
		// Store returns gRPC error codes, including not found.
		return err
	}
	defer blobReader.Close()
	if req.ReadOffset > blobReader.Digest().SizeBytes {
		return status.Errorf(codes.OutOfRange, "read offset larger than blob size")
	}
	if req.ReadOffset > 0 {
		if _, err := io.CopyN(ioutil.Discard, blobReader, req.ReadOffset); err != nil {
			return status.Errorf(codes.Internal, "failed seeking to offset")
		}
	}
	for {
		// TODO(mwitkow): This allocates a lot, try moving it to the top.
		chunkBuffer := make([]byte, *chunkSizeBytes)
		n, readErr := blobReader.Read(chunkBuffer)
		if readErr != nil && readErr != io.EOF {
			if statusErr, ok := status.FromError(readErr); ok {
				return statusErr.Err()
			} else {
				return status.Errorf(codes.DataLoss, "cannot read this file %v", readErr)
			}
		}
		if n > 0 {
			if err := readStream.Send(&bytestream.ReadResponse{Data: chunkBuffer[:n]}); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			break
		}
	}
	return nil
}

func (l *local) Write(writeStream bytestream.ByteStream_WriteServer) error {
	firstMsg, err := writeStream.Recv()
	if err != nil {
		return err
	}
	// TODO(mwitkow): Handle instance name of the request resourceName
	blobDigest, err := util.ResourcePathToContentDigest(firstMsg.ResourceName)
	if err != nil {
		return err
	}
	if firstMsg.WriteOffset > 0 {
		// TODO(mwitkow): Implement this write resumption. According to the docs, returning NotFound should be safe.
		return status.Errorf(codes.Unimplemented, "write resumption hasn't been implemented")
	}
	blobWriter, err := l.store.Write(writeStream.Context(), blobDigest)
	defer blobWriter.Close()
	writeChunk := firstMsg
	for true {
		if len(writeChunk.Data) > 0 {
			n, writeErr := blobWriter.Write(writeChunk.Data)
			if n != len(writeChunk.Data) {
				return status.Errorf(codes.Internal, "bad writer implementation, wrote partially %d of %d", n, len(writeChunk.Data))
			}
			if writeErr != nil {
				if statusErr, ok := status.FromError(writeErr); ok {
					return statusErr.Err()
				} else {
					return status.Errorf(codes.DataLoss, "cannot read this file %v", writeErr)
				}
			}
		}
		if writeChunk.FinishWrite == true {
			break
		}
		var err error
		writeChunk, err = writeStream.Recv()
		if err != nil {
			if err == io.EOF {
				return status.Errorf(codes.Unimplemented, "received an EOF without FinishWrite, resumption is not supported")
			} else {
				return err
			}
		}
	}
	return nil
}

func (l *local) QueryWriteStatus(context.Context, *bytestream.QueryWriteStatusRequest) (*bytestream.QueryWriteStatusResponse, error) {
	// TODO(mwitkow): Implement this write resumption. According to the docs, returning NotFound should be safe.
	return nil, status.Errorf(codes.NotFound, "write resumption is not supported")
}
