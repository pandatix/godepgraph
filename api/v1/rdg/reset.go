package apiv1rdg

import (
	"context"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (*RDG) Reset(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	global.Log().Info(ctx, "reseting RDG")
	return nil, reset(ctx, global.GetNeo4JManager())
}
