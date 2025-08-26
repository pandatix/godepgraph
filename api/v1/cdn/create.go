package apiv1cdn

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/pandatix/godepgraph/global"
)

func (*CDN) CreateLibrary(ctx context.Context, req *CreateLibraryRequest) (*Library, error) {
	logger := global.Log()
	span := trace.SpanFromContext(ctx)

	logger.Info(ctx, "creating library",
		zap.String("name", req.GetName()),
		zap.String("version", req.GetVersion()),
		zap.Bool("test", req.GetTest()),
	)

	ana := NewAnalyser(&AnalyserParams{
		Tests: req.GetTest(),
	})
	span.AddEvent("processing module")
	res, err := ana.Process(ctx, req.GetName(), req.GetVersion())
	if err != nil {
		logger.Error(ctx, "processing module",
			zap.Error(err),
		)
		return nil, err
	}
	span.AddEvent("processed module, exporting it")

	man := global.GetNeo4JManager()
	if err := res.Export(ctx, man); err != nil {
		return nil, err
	}
	return retrieveLibrary(ctx, man, req.GetName(), req.GetVersion())
}
