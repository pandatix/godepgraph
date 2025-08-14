package apiv1alg4

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/multierr"

	neo4jSvc "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/pkg/services/neo4j"
)

func upsertBinding(ctx context.Context, man *neo4jSvc.Manager, req *CreateBindingRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			// Match possible targets
			OPTIONAL MATCH (lib:Library {name: $libraryName, version: $libraryVersion})
			OPTIONAL MATCH (comp:Component {name: $componentName, version: $componentVersion})
			OPTIONAL MATCH (asset:Asset {name: $assetName, version: $assetVersion})

			// Check if a Binding already exists with the same relationships
			OPTIONAL MATCH (existing:Binding)
			WHERE 
				(lib IS NULL OR (existing)-[:SPECIALIZES_INTO]->(lib)) AND
				(comp IS NULL OR (existing)-[:SPECIALIZES_INTO]->(comp)) AND
				(asset IS NULL OR (existing)-[:SPECIALIZES_INTO]->(asset))
			WITH lib, comp, asset, existing
			WHERE existing IS NULL

			// Create new Binding and connect to the matched nodes
			CREATE (b:Binding)
			FOREACH (_ IN CASE WHEN lib IS NOT NULL THEN [1] ELSE [] END |
				MERGE (b)-[:SPECIALIZES_INTO]->(lib)
			)
			FOREACH (_ IN CASE WHEN comp IS NOT NULL THEN [1] ELSE [] END |
				MERGE (b)-[:SPECIALIZES_INTO]->(comp)
			)
			FOREACH (_ IN CASE WHEN asset IS NOT NULL THEN [1] ELSE [] END |
				MERGE (b)-[:SPECIALIZES_INTO]->(asset)
			)
			`,
			map[string]any{
				"libraryName":      req.GetLibrary().GetName(),
				"libraryVersion":   req.GetLibrary().GetVersion(),
				"componentName":    req.GetComponent().GetName(),
				"componentVersion": req.GetComponent().GetVersion(),
				"assetName":        req.GetAsset().GetName(),
				"assetVersion":     req.GetAsset().GetVersion(),
			},
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func upsertServes(ctx context.Context, man *neo4jSvc.Manager, req *CreateServesRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (s:Symbol {identity: $identity})
			MATCH (c:Component {name: $componentName, version: $componentVersion})<-[:EXPOSES]-(e:Endpoint {name: $endpointName})
			MERGE (e)-[:SERVES]->(s)
			`,
			map[string]any{
				"identity":         req.GetSymbol().GetIdentity(),
				"componentName":    req.GetEndpoint().GetExposes().GetName(),
				"componentVersion": req.GetEndpoint().GetExposes().GetVersion(),
				"endpointName":     req.GetEndpoint().GetName(),
			},
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}
