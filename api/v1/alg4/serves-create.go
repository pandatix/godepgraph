package apiv1alg4

import (
	"context"

	"github.com/pandatix/godepgraph/global"
	"go.uber.org/zap"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func (*Alg4) CreateServes(ctx context.Context, req *CreateServesRequest) (*emptypb.Empty, error) {
	logger := global.Log()
	man := global.GetNeo4JManager()

	logger.Info(ctx, "creating serves relationship",
		zap.String("symbol.identity", req.GetSymbol().GetIdentity()),
		zap.String("endpoint.name", req.GetEndpoint().GetName()),
		zap.String("component.name", req.GetEndpoint().GetExposes().GetName()),
		zap.String("component.version", req.GetEndpoint().GetExposes().GetVersion()),
	)

	return nil, upsertServes(ctx, man, req)
}
