package apiv1rdg

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/common"
	neo4jSvc "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/pkg/services/neo4j"
)

func (ana *Analysis) Export(ctx context.Context, man *neo4jSvc.Manager) error {
	span := trace.SpanFromContext(ctx)

	// Create all vertices
	span.AddEvent("upserting vertices")
	for _, as := range ana.Assets {
		if err := ana.upsertAsset(ctx, man, as); err != nil {
			return err
		}
	}
	// TODO export pairings

	return nil
}

func (ana *Analysis) upsertAsset(ctx context.Context, man *neo4jSvc.Manager, r *Resource) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// If the module already exist, don't duplicate it
		res, err := tx.Run(ctx,
			`
			MATCH (a:Asset)
			WHERE (a.id = $id OR $id IS NULL OR a.id IS NULL)
			RETURN COUNT(a) > 0 AS exists
			`,
			map[string]any{
				"id": r.ID,
			},
		)
		if err != nil {
			return nil, err
		}
		s, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}
		exist, _ := s.Get("exists")
		if (exist).(bool) {
			return nil, nil
		}

		// Else create it
		return tx.Run(ctx,
			"MERGE (a:Asset {id: $id})",
			map[string]any{
				"id": r.ID,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func reset(ctx context.Context, man *neo4jSvc.Manager) error {
	return multierr.Combine(
		common.Trash(ctx, man, `MATCH (n) WHERE n:Asset`, "n"),
	)
}
