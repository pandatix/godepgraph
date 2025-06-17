package sig

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/multierr"
)

func (ana *Analysis) Export(ctx context.Context, con DBConnector) error {
	fmt.Printf("Exporting SIG analysis results\n")

	// Create all vertices
	for _, comp := range ana.components {
		if err := con.UpsertComponent(ctx, comp); err != nil {
			return err
		}
	}

	// Create all edges (guarantee that all vertices exist)
	for _, comp := range ana.components {
		if err := con.UpsertInteractions(ctx, comp); err != nil {
			return err
		}
	}

	return nil
}

type DBConnector interface {
	// --- Vertices ---
	UpsertComponent(context.Context, *Component) error

	// --- Edges ---
	UpsertInteractions(context.Context, *Component) error
}

// region Neo4J

type Neo4JConnector struct {
	driver neo4j.DriverWithContext
}

func NewNeo4JConnector(ctx context.Context, url string) (*Neo4JConnector, error) {
	driver, err := neo4j.NewDriverWithContext(url, neo4j.NoAuth())
	if err != nil {
		return nil, err
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, err
	}

	return &Neo4JConnector{
		driver: driver,
	}, nil
}

var _ DBConnector = (*Neo4JConnector)(nil)

func (con *Neo4JConnector) UpsertComponent(ctx context.Context, comp *Component) error {
	session := con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// If the module already exist, don't duplicate it
		res, err := tx.Run(ctx,
			`
			MATCH (c:Component)
			WHERE (c.name = $name OR $name IS NULL OR c.name IS NULL)
			RETURN COUNT(c) > 0 AS exists
			`,
			map[string]any{
				"name": comp.Name,
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
			"MERGE (c:Component {name: $name})",
			map[string]any{
				"name": comp.Name,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func (con *Neo4JConnector) UpsertInteractions(ctx context.Context, comp *Component) (merr error) {
	for _, it := range comp.Interactions {
		merr = multierr.Append(merr, con.upsertInteraction(ctx, comp, it))
	}
	return
}

func (con *Neo4JConnector) upsertInteraction(ctx context.Context, comp *Component, it *Interaction) error {
	session := con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (from:Component {name: $fromName})
			MATCH (to:Component {name: $toName})
			MERGE (i:Interaction {timestamp: $timestamp})
			ON CREATE SET i.timestamp = $timestamp
			ON CREATE SET i.name = $iname
			MERGE (from)-[:INTERACTS]->(i)
			MERGE (i)-[:TO]->(to)
			RETURN NULL;
			`,
			map[string]any{
				"fromName":  comp.Name,
				"toName":    it.To,
				"timestamp": it.Timestamp,
				"iname":     it.Name,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}
