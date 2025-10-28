package apiv1sig

import (
	"context"

	"github.com/pandatix/godepgraph/global"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func (*SIG) QueryComponent(ctx context.Context, _ *emptypb.Empty) (*Components, error) {
	logger := global.Log()

	logger.Info(ctx, "querying components")

	return queryComponents(ctx, global.GetNeo4JManager())
}
