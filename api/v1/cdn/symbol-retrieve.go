package apiv1cdn

import (
	"context"

	"go.uber.org/zap"

	"github.com/pandatix/godepgraph/global"
)

func (*CDN) RetrieveSymbolCallGraphDependencies(ctx context.Context, req *RetrieveSymbolCallGraphDependenciesRequest) (*SymbolDepGraph, error) {
	logger := global.Log()
	man := global.GetNeo4JManager()

	logger.Info(ctx, "retrieving call-graph dependencies",
		zap.String("symbol", req.GetSymbol().GetIdentity()),
	)

	return retrieveSymbolCallGraphDependencies(ctx, man, req.GetSymbol())
}
