package apiv1alg4

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/multierr"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
	neo4jSvc "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/pkg/services/neo4j"
)

func init() {
	global.RegisterNeo4JInitializer(func(ctx context.Context, man *neo4jSvc.Manager) error {
		session, err := man.NewSession(ctx)
		if err != nil {
			return err
		}

		_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx,
				`
				CREATE CONSTRAINT vulnerability_unique IF NOT EXISTS
				FOR (v:Vulnerability)
				REQUIRE v.identity IS UNIQUE
				`,
				nil,
			)
			return nil, err
		})
		return multierr.Append(err, session.Close(ctx))
	})
}

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

func upsertVulnerability(ctx context.Context, man *neo4jSvc.Manager, req *CreateVulnerabilityRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// If the vulnerability already exist, don't duplicate it
		res, err := tx.Run(ctx,
			`
			MATCH (v:Vulnerability)
			WHERE (v.identity = $identity OR $identity IS NULL OR v.identity IS NULL)
			RETURN count(v) > 0 AS exists
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
		if err != nil {
			return nil, err
		}
		s, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}
		exists, _ := s.Get("exists")
		if exists, ok := (exists).(bool); ok && exists {
			return nil, nil
		}

		// Else create it
		return tx.Run(ctx,
			`
			MATCH (s:Symbol {identity: $threatens})
			SET s.mark = true
			WITH s
			MERGE (v:Vulnerability {identity: $identity})
			MERGE (v)-[:THREATENS]->(s)
			`,
			map[string]any{
				"identity":  req.GetIdentity(),
				"threatens": req.GetTreatens(),
			},
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func allReachingSymbols(ctx context.Context, man *neo4jSvc.Manager) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	// One query should be too complex to write and maintain, use a fixed-point strategy
	for {
		res, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			res, err := tx.Run(ctx,
				`
				MATCH (s:Symbol {mark: true})
				MATCH (s2:Symbol)<-[:CALLER]-(a:CallGraphDependency)-[:CALLEES]->(s)
				WHERE s2.mark <> true OR s2.mark IS NULL
				SET a.mark = true
				SET s2.mark = true
				RETURN count(*) AS newlyMarked
				`,
				nil,
			)
			if err != nil {
				return nil, err
			}
			s, err := res.Single(ctx)
			if err != nil {
				return nil, err
			}
			nm, _ := s.Get("newlyMarked")
			stop := nm.(int64) == 0
			return stop, nil
		})
		if err != nil {
			return err
		}
		if res.(bool) {
			break
		}
	}

	return multierr.Combine(err, session.Close(ctx))
}

func allProvidingLibraries(ctx context.Context, man *neo4jSvc.Manager) (merr error) {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	// Mark Libraries
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (l:Library)
			MATCH (l)-[:PROVIDES]->(s:Symbol {mark: true})
			WITH DISTINCT l
			SET l.mark = true
			`,
			nil,
		)
	})
	merr = multierr.Append(merr, err)

	// Then mark specializations (3.a and 4.a)
	// => Component->Library
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (l:Library {mark: true})
			MATCH (b:Binding)-[:SPECIALIZES_INTO]->(l)
			MATCH (b)-[:SPECIALIZES_INTO]->(c:Component)
			WITH DISTINCT b, c
			SET b.mark = true
			SET c.mark = true
			`,
			nil,
		)
	})
	merr = multierr.Append(merr, err)

	// => Asset->Library
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (l:Library {mark: true})
			MATCH (b:Binding)-[:SPECIALIZES_INTO]->(l)
			MATCH (b)-[:SPECIALIZES_INTO]->(a:Asset)
			WITH DISTINCT b, a
			SET b.mark = true
			SET a.mark = true
			`,
			nil,
		)
	})
	merr = multierr.Append(merr, err)

	// => Asset->Component (could happen if the Library has not been handled first, thus the specialization
	// can only comes from a Component)
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (c:Component {mark: true})
			MATCH (b:Binding)-[:SPECIALIZES_INTO]->(c)
			MATCH (b)-[:SPECIALIZES_INTO]->(a:Asset)
			WITH DISTINCT b, a
			SET b.mark = true
			SET a.mark = true
			`,
			nil,
		)
	})
	merr = multierr.Append(merr, err)

	return multierr.Combine(merr, session.Close(ctx))
}

func allReachingComponents(ctx context.Context, man *neo4jSvc.Manager) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (s:Symbol {mark: true})
			// Find all Endpoints serving vulnerable Symbols
			MATCH (e:Endpoint)-[:SERVES]->(s)
			SET e.mark = true

			// Find NetworkDependencies that call these Endpoints
			WITH e
			OPTIONAL MATCH (d:InterComponentDependency)-[:CALLEES]->(e)
			SET d.mark = true

			// Find Components exposed by Endpoints called by these NetworkDependencies
			WITH d
			OPTIONAL MATCH (d)-[:CALLER]->(e2:Endpoint)
			SET e2.mark = true

			WITH e2
			OPTIONAL MATCH (e2)-[:EXPOSES]->(c:Component)
			SET c.mark = true
			`,
			nil,
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func allHostingAssets(ctx context.Context, man *neo4jSvc.Manager) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (c:Component {mark: true})
			MATCH (a:Asset)<-[:HOSTED_BY]-(c)
			SET a.mark = true
			`,
			nil,
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func lateralMovement(ctx context.Context, man *neo4jSvc.Manager) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (a:Asset{mark: true})
			MATCH (c:Component)-[:HOSTED_BY]->(a)
			SET c.mark = true
			`,
			nil,
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func allSystems(ctx context.Context, man *neo4jSvc.Manager) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (c:Component{mark: true})
			MATCH (s:System)-[:COMPOSED_OF]->(c)
			SET s.mark = true
			`,
			nil,
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}
