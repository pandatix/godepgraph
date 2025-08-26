package apiv1sig

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
)

func (*SIG) CreateInterComponentDependency(ctx context.Context, req *CreateInterComponentDependencyRequest) (*emptypb.Empty, error) {
	logger := global.Log()
	man := global.GetNeo4JManager()

	logger.Info(ctx, "creating inter-component dependency",
		zap.String("caller", req.Caller.Name),
	)

	// Create the origin
	if err := upsertEndpoint(ctx, man, req.GetCaller().GetExposes(), req.GetCaller().GetName()); err != nil {
		return nil, err
	}

	// Then all destinations, and link them
	var merr error
	for _, callee := range req.GetCallees() {
		if err := upsertEndpoint(ctx, man, callee.GetExposes(), callee.GetName()); err != nil {
			merr = multierr.Append(merr, err)
			continue // best effort
		}
	}
	if err := upsertInterComponentDependencies(ctx, man, req); err != nil {
		merr = multierr.Append(merr, err)
	}
	if merr != nil {
		return nil, merr
	}

	return nil, nil
}
