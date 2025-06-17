package cdn

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/multierr"
)

func (ana *Analysis) Export(ctx context.Context, con DBConnector, from *Module) error {
	fmt.Printf("Exporting CDN analysis results\n")

	// Create all vertices
	for _, mod := range ana.modules {
		if err := con.UpsertModule(ctx, mod); err != nil {
			return err
		}
		for _, pkg := range mod.Packages {
			pkg = ana.packages[pkg.Name]
			if err := con.UpsertPackage(ctx, pkg); err != nil {
				return err
			}
			for _, f := range pkg.Functions {
				if err := con.RecurseUpsertFunction(ctx, ana, f); err != nil {
					return err
				}
			}
		}
	}

	// Create all module and package edges (guarantee that all vertices exist)
	for _, mod := range ana.modules {
		for _, pkg := range mod.Packages {
			pkg = ana.packages[pkg.Name]
			if err := con.UpsertContains(ctx, mod, pkg); err != nil {
				return err
			}

			for _, f := range pkg.Functions {
				if err := con.UpsertDefines(ctx, pkg, f); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// DBConnector defines the methods required for a database driver to
// implement in order for the Call-graph Dependency Network to be persisted.
type DBConnector interface {
	// --- Vertices ---

	// UpsertModule creates the [*Module] if does not exist yet,
	// else do nothing.
	UpsertModule(context.Context, *Module) error
	// UpsertPackage creates the [*Package] if does not exist yet,
	// else do nothing.
	UpsertPackage(context.Context, *Package) error
	// RecurseUpsertFunction creates the [*Function] if does not exist yet,
	// else do nothing, and propagates to the dependencies i.e. the
	// function it calls.
	RecurseUpsertFunction(context.Context, *Analysis, *Function) error

	// --- Edges ---

	// UpsertContains creates the [*Module]-[:contains]->[*Package]
	// edge if does not exist yet, else do nothing.
	UpsertContains(context.Context, *Module, *Package) error
	// UpsertDefines creates the [*Package]-[:defines]->[*Function]
	// edge if does not exist yet, else do nothing.
	UpsertDefines(context.Context, *Package, *Function) error
	// UpsertCalls creates the [*Function]-[:calls]->[*Function]
	// edge if does not exist yet, else do nothing.
	UpsertCalls(context.Context, *Function, *Function) error
}

// region Neo4J

type Neo4JConnector struct {
	driver neo4j.DriverWithContext
}

// NewNeo4JConnector opens a connection to the Neo4J database provided
// its URL, and check connectivity.
// It then could be used as a Call-graph Dependency Network connector
// to export the results.
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

func (con *Neo4JConnector) UpsertModule(ctx context.Context, mod *Module) error {
	session := con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// If the module already exist, don't duplicate it
		res, err := tx.Run(ctx,
			`
			MATCH (m:Module)
			WHERE (m.name = $name OR $name IS NULL OR m.name IS NULL)
			RETURN COUNT(m) > 0 AS exists
			`,
			map[string]any{
				"name": mod.Name,
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
			"MERGE (m:Module {name: $name, version: $version})",
			map[string]any{
				"name":    mod.Name,
				"version": mod.Version,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func (con *Neo4JConnector) UpsertPackage(ctx context.Context, pkg *Package) error {
	session := con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// If the module already exist, don't duplicate it
		res, err := tx.Run(ctx,
			`
			MATCH (p:Package)
			WHERE (p.name = $name OR $name IS NULL OR p.name IS NULL)
			RETURN COUNT(p) > 0 AS exists
			`,
			map[string]any{
				"name": pkg.Name,
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
			"MERGE (p:Package {name: $name})",
			map[string]any{
				"name": pkg.Name,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func (con *Neo4JConnector) UpsertContains(ctx context.Context, mod *Module, pkg *Package) error {
	session := con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (from:Module {name: $modName})
			MATCH (to:Package)
			WHERE (to.name = $pkgName OR $pkgName IS NULL OR to.name IS NULL)
			MERGE (from)-[:CONTAINS]->(to)
			`,
			map[string]any{
				"modName": mod.Name,
				"pkgName": pkg.Name,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func (con *Neo4JConnector) UpsertDefines(ctx context.Context, pkg *Package, f *Function) error {
	session := con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (from:Package {name: $pkgName})
			MATCH (to:Function)
			WHERE (to.identity = $identity OR $identity IS NULL OR to.identity IS NULL)
			MERGE (from)-[:DEFINES]->(to)
			`,
			map[string]any{
				"pkgName":  pkg.Name,
				"identity": f.Identity,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func (con *Neo4JConnector) UpsertCalls(ctx context.Context, from *Function, to *Function) error {
	session := con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			MATCH (from:Function {identity: $fromIdentity})
			MATCH (to:Function)
			WHERE (to.identity = $toIdentity OR $toIdentity IS NULL OR to.identity IS NULL)
			MERGE (from)-[:CALLS]->(to)
			`,
			map[string]any{
				"fromIdentity": from.Identity,
				"toIdentity":   to.Identity,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func (con *Neo4JConnector) RecurseUpsertFunction(ctx context.Context, ana *Analysis, from *Function) error {
	// If the function is already covered, there is no need at creating it
	// and its dependencies.
	session := con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	exist, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx,
			`
			MATCH (f:Function)
			WHERE (f.identity = $identity OR $identity IS NULL OR f.identity IS NULL)
			RETURN COUNT(f) > 0 AS exists
			`,
			map[string]any{
				"identity": from.Identity,
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
		return exist, nil
	})
	multierr.Append(err, session.Close(ctx))
	if err != nil {
		return err
	}
	if exist.(bool) {
		return nil
	}

	// Then create it
	session = con.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// If the module already exist, don't duplicate it
		res, err := tx.Run(ctx,
			`
			MATCH (f:Function)
			WHERE (f.identity = $identity OR $identity IS NULL OR f.identity IS NULL)
			RETURN COUNT(f) > 0 AS exists
			`,
			map[string]any{
				"identity": from.Identity,
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
			"MERGE (f:Function {identity: $identity})",
			map[string]any{
				"identity": from.Identity,
			},
		)
	})
	err = multierr.Append(err, session.Close(ctx))
	if err != nil {
		return err
	}

	// And recurse on the dependencies
	for pkg, funcs := range from.Dependencies {
		for fname := range funcs {
			dep := ana.packages[pkg].Functions[fname]
			if err := con.RecurseUpsertFunction(ctx, ana, dep); err != nil {
				return err
			}
			if err := con.UpsertCalls(ctx, from, dep); err != nil {
				return err
			}
		}
	}
	return nil
}
