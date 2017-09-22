package actioncache

import (
	"github.com/mwitkow/bazel-distcache/stores/action"
	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type local struct {
	store action.Store
}

func (l *local) GetActionResult(ctx context.Context, req remoteexecution.GetActionResultRequest) (*remoteexecution.ActionResult, error) {
	if req.GetActionDigest() == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "action digest must be set")
	}
	// TODO(mwitkow): Handle req.InstanceName
	actionResult, err := l.store.Get(req.GetActionDigest())
	// errors from storage are gRPC so we're good.
	return actionResult, err

}

func (l *local) UpdateActionResult(ctx context.Context, req *remoteexecution.UpdateActionResultRequest) (*remoteexecution.ActionResult, error) {
	// TODO(mwitkow): Handle req.InstanceName
	if req.GetActionDigest() == nil || req.GetActionResult() == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "action result and dugest must be set")
	}
	err := l.store.Store(req.ActionDigest, req.ActionResult)
	if err != nil {
		// errors from storage are gRPC so we're good.
		return nil, err
	}
	return req.ActionResult, nil
}
