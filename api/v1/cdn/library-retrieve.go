package apiv1cdn

import (
	"context"

	"go.uber.org/zap"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
)

func (*CDN) RetrieveLibrary(ctx context.Context, req *RetrieveLibraryRequest) (*Library, error) {
	logger := global.Log()

	logger.Info(ctx, "getting library",
		zap.String("name", req.GetName()),
		zap.String("version", req.GetVersion()),
	)

	return retrieveLibrary(ctx, global.GetNeo4JManager(), req.GetName(), req.GetVersion())
}
