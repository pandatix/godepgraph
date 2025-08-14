package apiv1sig

import (
	"context"
	"errors"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
	"go.uber.org/multierr"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/common"
	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
	neo4jSvc "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/pkg/services/neo4j"
)

func init() {
	global.RegisterNeo4JInitializer(func(ctx context.Context, man *neo4jSvc.Manager) (merr error) {
		session, err := man.NewSession(ctx)
		if err != nil {
			return err
		}
		defer session.Close(ctx)

		_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx,
				`
				CREATE CONSTRAINT component_unique IF NOT EXISTS
				FOR (c:Component)
				REQUIRE (c.name, c.version) IS UNIQUE
				`,
				nil,
			)
			return nil, err
		})
		merr = multierr.Append(merr, err)

		return
	})
}

func upsertComponent(ctx context.Context, man *neo4jSvc.Manager, name, version string) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// If the module already exist, don't duplicate it
		res, err := tx.Run(ctx,
			`
			MATCH (c:Component)
			WHERE (c.name = $name OR $name IS NULL OR c.name IS NULL)
				AND (c.version = $version OR $version IS NULL OR c.version IS NULL)
			RETURN COUNT(c) > 0 AS exists
			`,
			map[string]any{
				"name":    name,
				"version": version,
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
			"MERGE (c:Component {name: $name, version: $version})",
			map[string]any{
				"name":    name,
				"version": version,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func retrieveComponent(ctx context.Context, man *neo4jSvc.Manager, name, version string) (*Component, error) {
	session, err := man.NewSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx,
			`
			MATCH (c:Component {name: $name, version: $version})
			OPTIONAL MATCH (c)<-[:EXPOSES]-(e:Endpoint)
			RETURN c, collect(e) AS endpoints
			`,
			map[string]any{
				"name":    name,
				"version": version,
			},
		)
		if err != nil {
			return nil, err
		}
		if !res.Next(ctx) {
			return nil, errors.New("component not found")
		}
		rec := res.Record()

		edpItfs := rec.Values[1].([]any)
		edps := make([]string, 0, len(edpItfs))
		for _, edpIt := range edpItfs {
			edps = append(edps, edpIt.(dbtype.Node).Props["name"].(string))
		}
		return &Component{
			Name:      name,
			Version:   version,
			Endpoints: edps,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return res.(*Component), nil
}

func upsertEndpoint(ctx context.Context, man *neo4jSvc.Manager, component *CreateNetworkDependencyEndpointComponentRequest, edp string) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (c:Component {name: $componentName, version: $componentVersion})
			OPTIONAL MATCH (e:Endpoint {name: $edpName})-[:EXPOSES]->(c)

			FOREACH (_ IN CASE WHEN e IS NULL THEN [1] ELSE [] END |
				CREATE (n:Endpoint{name: $edpName})
				CREATE (n)-[:EXPOSES]->(c)
			)
			`,
			map[string]any{
				"componentName":    component.GetName(),
				"componentVersion": component.GetVersion(),
				"edpName":          edp,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func upsertNetworkDependencies(ctx context.Context, man *neo4jSvc.Manager, req *CreateNetworkDependencyRequest) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			// Find caller component and endpoint
			MATCH (endpointCaller:Endpoint{name: $callerEdpName})-[:EXPOSES]->(componentCaller:Component{name: $callerComponentName, version: $callerComponentVersion})

			UNWIND $callees AS callee
				// Find callee component and endpoint
				MATCH (endpointCallee:Endpoint {name: callee.edpName})-[:EXPOSES]->(:Component {name: callee.componentName, version: callee.componentVersion})

				// Merge a NetworkDependency uniquely identified by caller and callee
				MERGE (endpointCaller)<-[:CALLER]-(:NetworkDependency)-[:CALLEES]->(endpointCallee)
			`,
			map[string]any{
				"callerEdpName":          req.GetCaller().GetName(),
				"callerComponentName":    req.GetCaller().GetExposes().GetName(),
				"callerComponentVersion": req.GetCaller().GetExposes().GetVersion(),
				"callees": func() []map[string]string {
					out := make([]map[string]string, 0, len(req.GetCallees()))
					for _, callee := range req.GetCallees() {
						out = append(out, map[string]string{
							"edpName":          callee.GetName(),
							"componentName":    callee.GetExposes().GetName(),
							"componentVersion": callee.GetExposes().GetVersion(),
						})
					}
					return out
				}(),
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func reset(ctx context.Context, man *neo4jSvc.Manager) error {
	return multierr.Combine(
		common.Trash(ctx, man, `MATCH (:Component)<-[r:EXPOSES]-(:Endpoint)`, "r"),
		common.Trash(ctx, man, `MATCH (:Endpoint)<-[r:CALLER]-(:NetworkDependency)`, "r"),
		common.Trash(ctx, man, `MATCH (:Endpoint)<-[r:CALLEES]-(:NetworkDependency)`, "r"),
		common.Trash(ctx, man, `MATCH (n) WHERE n:Component OR n:Endpoint OR n:NetworkDependency`, "n"),
	)
}
