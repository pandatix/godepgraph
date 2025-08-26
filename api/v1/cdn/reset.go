package apiv1cdn

import (
	"context"

	emptypb "google.golang.org/protobuf/types/known/emptypb"

	"github.com/pandatix/godepgraph/global"
)

func (*CDN) Reset(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	global.Log().Info(ctx, "reseting CDN")
	return nil, reset(ctx, global.GetNeo4JManager())
}
