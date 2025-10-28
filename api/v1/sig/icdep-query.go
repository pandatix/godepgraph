package apiv1sig

import (
	"context"

	"github.com/pandatix/godepgraph/global"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func (*SIG) QueryInterComponentDependency(ctx context.Context, _ *emptypb.Empty) (*InterComponentDependencies, error) {
	logger := global.Log()
	man := global.GetNeo4JManager()

	logger.Info(ctx, "querying inter-component dependencies")

	return queryICDS(ctx, man)
}
