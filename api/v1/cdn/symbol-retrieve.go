package apiv1cdn

import (
	"context"

	"go.uber.org/zap"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
)

func (*CDN) RetrieveSymbolASTDependencies(ctx context.Context, req *RetrieveSymbolASTDependenciesRequest) (*SymbolDepGraph, error) {
	logger := global.Log()
	man := global.GetNeo4JManager()

	logger.Info(ctx, "retrieving AST dependencies",
		zap.String("symbol", req.GetSymbol().GetIdentity()),
	)

	return retrieveSymbolASTDependencies(ctx, man, req.GetSymbol())
}
