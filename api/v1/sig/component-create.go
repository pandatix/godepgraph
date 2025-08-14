package apiv1sig

import (
	"context"

	"go.uber.org/zap"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
)

func (*SIG) CreateComponent(ctx context.Context, req *CreateComponentRequest) (*Component, error) {
	logger := global.Log()
	man := global.GetNeo4JManager()

	logger.Info(ctx, "creating component",
		zap.String("name", req.GetName()),
		zap.String("version", req.GetVersion()),
	)

	if err := upsertComponent(ctx, man, req.GetName(), req.GetVersion()); err != nil {
		return nil, err
	}

	return retrieveComponent(ctx, man, req.GetName(), req.GetVersion())
}
