package apiv1sig

import (
	"context"

	emptypb "google.golang.org/protobuf/types/known/emptypb"

	"github.com/pandatix/godepgraph/global"
)

func (*SIG) Reset(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	global.Log().Info(ctx, "reseting SIG")
	return nil, reset(ctx, global.GetNeo4JManager())
}
