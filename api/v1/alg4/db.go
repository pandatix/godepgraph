package apiv1alg4

import (
	"context"
	"errors"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/multierr"

	"github.com/pandatix/godepgraph/global"
	neo4jSvc "github.com/pandatix/godepgraph/pkg/services/neo4j"
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
			MERGE (v:Vulnerability {identity: $identity})
			MERGE (v)-[:THREATENS]->(s)
			MERGE (v)-[:MARKS]->(s)
			`,
			map[string]any{
				"identity":  req.GetIdentity(),
				"threatens": req.GetTreatens(),
			},
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func retrieveVulnerability(ctx context.Context, man *neo4jSvc.Manager, identity string) (*Vulnerability, error) {
	session, err := man.NewSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx,
			`
			MATCH (v:Vulnerability {identity: $identity})
			OPTIONAL MATCH (s:Symbol)<-[:THREATENS]-(v)
			RETURN s
			`, // TODO catch all other labeled nodes that are marked by this vulnerability
			map[string]any{
				"identity": identity,
			},
		)
		if err != nil {
			return nil, err
		}
		if !res.Next(ctx) {
			return nil, errors.New("vulnerability not found")
		}
		rec := res.Record()

		symNode := rec.Values[0].(neo4j.Node)

		return &Vulnerability{
			Identity: identity,
			Treatens: symNode.Props["identity"].(string),
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return res.(*Vulnerability), nil
}

func deleteVulnerability(ctx context.Context, man *neo4jSvc.Manager, req *DeleteVulnerabilityRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx,
			`
		MATCH (v:Vulnerability{identity: $identity})-[t:THREATENS]->(s)
		OPTIONAL MATCH (v)-[m:MARKS]->(n)
		DELETE t, m, v
		`, map[string]any{
				"identity": req.GetIdentity(),
			})
		return nil, err
	})
	return err
}

func allReachingSymbols(ctx context.Context, man *neo4jSvc.Manager, req *CreateVulnerabilityRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	// One query should be too complex to write and maintain, use a fixed-point strategy
	for {
		res, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			res, err := tx.Run(ctx,
				`
				MATCH (v:Vulnerability{identity: $identity})
				MATCH (s:Symbol)<-[:MARKS]-(v)
				MATCH (s2:Symbol)<-[:CALLER]-(a:CallGraphDependency)-[:CALLEES]->(s)
				WHERE NOT (v)-[:MARKS]->(s2)
				MERGE (v)-[:MARKS]->(s2)
				RETURN count(s2) AS newlyMarked
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

func allProvidingLibraries(ctx context.Context, man *neo4jSvc.Manager, req *CreateVulnerabilityRequest) (merr error) {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	// Mark Libraries
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (v:Vulnerability{identity: $identity})
			MATCH (l:Library)-[:PROVIDES]->(s:Symbol)<-[:MARKS]-(v)
			WITH DISTINCT v, l
			MERGE (v)-[:MARKS]->(l)
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
	})
	merr = multierr.Append(merr, err)

	// Then mark specializations (3.a and 4.a)
	// => Component->Library
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (v:Vulnerability{identity: $identity})
			MATCH (l:Library)<-[:MARKS]-(v)
			MATCH (b:Binding)-[:SPECIALIZES_INTO]->(l)
			MATCH (b)-[:SPECIALIZES_INTO]->(c:Component)
			WITH DISTINCT v, b, c
			MERGE (v)-[:MARKS]->(b)
			MERGE (v)-[:MARKS]->(c)
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
	})
	merr = multierr.Append(merr, err)

	// => Asset->Library
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (v:Vulnerability{identity: $identity})
			MATCH (l:Library)<-[:MARKS]-(v)
			MATCH (b:Binding)-[:SPECIALIZES_INTO]->(l)
			MATCH (b)-[:SPECIALIZES_INTO]->(a:Asset)
			WITH DISTINCT v, b, a
			MERGE (v)-[:MARKS]->(b)
			MERGE (v)-[:MARKS]->(a)
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
	})
	merr = multierr.Append(merr, err)

	// => Asset->Component (could happen if the Library has not been handled first, thus the specialization
	// can only comes from a Component)
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (v:Vulnerability{identity: $identity})
			MATCH (c:Component)<-[:MARKS]-(v)
			MATCH (b:Binding)-[:SPECIALIZES_INTO]->(c)
			MATCH (b)-[:SPECIALIZES_INTO]->(a:Asset)
			WITH DISTINCT v, b, a
			MERGE (v)-[:MARKS]->(b)
			MERGE (v)-[:MARKS]->(a)
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
	})
	merr = multierr.Append(merr, err)

	return multierr.Combine(merr, session.Close(ctx))
}

func allReachingComponents(ctx context.Context, man *neo4jSvc.Manager, req *CreateVulnerabilityRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (v:Vulnerability{identity: $identity})

			MATCH (s:Symbol)<-[:MARKS]-(v)
			// Find all Endpoints serving vulnerable Symbols
			MATCH (e:Endpoint)-[:SERVES]->(s)
			MERGE (v)-[:MARKS]->(e)

			// Find NetworkDependencies that call these Endpoints
			WITH v, e
			OPTIONAL MATCH (d:InterComponentDependency)-[:CALLEES]->(e)
			MERGE (v)-[:MARKS]->(d)

			// Find Components exposed by Endpoints called by these NetworkDependencies
			WITH v, d
			OPTIONAL MATCH (d)-[:CALLER]->(e2:Endpoint)
			MERGE (v)-[:MARKS]->(e2)

			WITH v, e2
			OPTIONAL MATCH (e2)-[:EXPOSES]->(c:Component)
			MERGE (v)-[:MARKS]->(c)
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func allHostingAssets(ctx context.Context, man *neo4jSvc.Manager, req *CreateVulnerabilityRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (v:Vulnerability{identity: $identity})
			MATCH (c:Component)<-[:MARKS]-(s)
			MATCH (a:Asset)<-[:HOSTED_BY]-(c)
			MERGE (v)-[:MARKS]->(a)
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func lateralMovement(ctx context.Context, man *neo4jSvc.Manager, req *CreateVulnerabilityRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (v:Vulnerability{identity: $identity})
			MATCH (a:Asset)<-[:MARKS]-(v)
			MATCH (c:Component)-[:HOSTED_BY]->(a)
			MERGE (v)-[:MARKS]->(c)
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}

func allSystems(ctx context.Context, man *neo4jSvc.Manager, req *CreateVulnerabilityRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (v:Vulnerability{identity: $identity})
			MATCH (c:Component)<-[:MARKS]-(v)
			MATCH (s:System)-[:COMPOSED_OF]->(c)
			MERGE (v)-[:MARKS]->(s)
			`,
			map[string]any{
				"identity": req.GetIdentity(),
			},
		)
	})
	return multierr.Combine(err, session.Close(ctx))
}
