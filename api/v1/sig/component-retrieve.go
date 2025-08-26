package apiv1sig

import (
	"context"

	"go.uber.org/zap"

	"github.com/pandatix/godepgraph/global"
)

func (*SIG) RetrieveComponent(ctx context.Context, req *RetrieveComponentRequest) (*Component, error) {
	logger := global.Log()

	logger.Info(ctx, "getting component",
		zap.String("name", req.GetName()),
		zap.String("version", req.GetVersion()),
	)

	return retrieveComponent(ctx, global.GetNeo4JManager(), req.GetName(), req.GetVersion())
}
