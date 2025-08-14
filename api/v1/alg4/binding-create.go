package apiv1alg4

import (
	"context"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
)

func (*Alg4) CreateBinding(ctx context.Context, req *CreateBindingRequest) (*Binding, error) {
	logger := global.Log()
	man := global.GetNeo4JManager()

	logger.Info(ctx, "upserting binding")

	if err := upsertBinding(ctx, man, req); err != nil {
		return nil, err
	}
	return &Binding{
		Library: &LibraryOrRefinement{
			Name:    req.GetLibrary().GetName(),
			Version: req.GetLibrary().GetVersion(),
		},
		Component: &LibraryOrRefinement{
			Name:    req.GetComponent().GetName(),
			Version: req.GetComponent().GetVersion()},
		Asset: &LibraryOrRefinement{
			Name:    req.GetAsset().GetName(),
			Version: req.GetAsset().GetVersion()},
	}, nil
}
