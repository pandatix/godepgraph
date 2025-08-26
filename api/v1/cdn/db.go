package apiv1cdn

import (
	"context"
	"errors"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
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
				CREATE CONSTRAINT library_unique IF NOT EXISTS
				FOR (l:Library)
				REQUIRE (l.name, l.version) IS UNIQUE
				`,
				nil,
			)
			return nil, err
		})
		merr = multierr.Append(merr, err)

		_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx,
				`
				CREATE CONSTRAINT symbol_identity_unique IF NOT EXISTS
				FOR (s:Symbol)
				REQUIRE s.identity IS UNIQUE
				`,
				nil,
			)
			return nil, err
		})
		merr = multierr.Append(merr, err)

		return
	})
}

func (ana *Analysis) Export(ctx context.Context, man *neo4jSvc.Manager) error {
	// Extract all packages not provided by a library
	pkgCore := []*pkg{}
	for _, p := range ana.packages {
		var lib *library
		for _, l := range ana.libraries {
			if strings.HasPrefix(p.Name, l.Name) {
				lib = l
				break
			}
		}
		if lib == nil {
			pkgCore = append(pkgCore, p)
		}
	}
	// Upsert core library and add all symbols
	coreLib := &library{
		Name:     "Go",
		Version:  ana.goVersion,
		Packages: pkgCore,
	}
	coreExists, err := libraryExists(ctx, man, coreLib)
	if err != nil {
		return err
	}
	if !coreExists {
		if err := upsertLibrary(ctx, man, coreLib); err != nil {
			return err
		}
		if err := bulkUpsertSymbols(ctx, man, coreLib); err != nil {
			return err
		}
		if err := bulkUpsertCallGraphDependencies(ctx, man, coreLib); err != nil {
			return err
		}
	}

	// Then create all required nodes
	createdLibs := map[string]struct{}{}
	for lname, lib := range ana.libraries {
		// Don't need to upsert the library and its symbols if already exist
		libExists, err := libraryExists(ctx, man, lib)
		if err != nil {
			return err
		}
		if libExists {
			continue
		}
		createdLibs[lname] = struct{}{}

		if err := upsertLibrary(ctx, man, lib); err != nil {
			return err
		}

		// Then bulk upsert all symbols
		if err := bulkUpsertSymbols(ctx, man, lib); err != nil {
			return err
		}
	}

	// Finally create all relationships
	for lname := range createdLibs {
		lib := ana.libraries[lname]
		if err := bulkUpsertCallGraphDependencies(ctx, man, lib); err != nil {
			return err
		}
	}

	return nil
}

func libraryExists(ctx context.Context, man *neo4jSvc.Manager, lib *library) (bool, error) {
	session, err := man.NewSession(ctx)
	if err != nil {
		return false, err
	}
	exists, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx,
			`
			MATCH (l:Library)
			WHERE (l.name = $name OR $name IS NULL OR l.name IS NULL)
				AND (l.version = $version OR $version IS NULL OR l.version IS NULL)
			RETURN COUNT(l) > 0 AS exists
			`,
			map[string]any{
				"name":    lib.Name,
				"version": lib.Version,
			},
		)
		if err != nil {
			return nil, err
		}
		s, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}
		e, _ := s.Get("exists")
		exists, ok := (e).(bool)
		if !ok {
			panic("invalid type")
		}
		return exists, nil
	})
	return exists.(bool), err
}

func bulkUpsertSymbols(ctx context.Context, man *neo4jSvc.Manager, lib *library) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx,
			`
			UNWIND $symbols AS sym
				CREATE (s:Symbol {identity: sym.identity})
				WITH s
				MATCH (lib:Library {name: $name, version: $version})
				MERGE (lib)-[:PROVIDES]->(s)
			`,
			map[string]any{
				"name":    lib.Name,
				"version": lib.Version,
				"symbols": func() []map[string]string {
					symbols := []map[string]string{}
					for _, pkg := range lib.Packages {
						for _, sym := range pkg.Symbols {
							symbols = append(symbols, map[string]string{
								"identity": sym.Identity,
							})
						}
					}
					return symbols
				}(),
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func upsertLibrary(ctx context.Context, man *neo4jSvc.Manager, lib *library) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// If the library already exist, don't duplicate it
		res, err := tx.Run(ctx,
			`
			MATCH (l:Library)
			WHERE (l.name = $name OR $name IS NULL OR l.name IS NULL)
				AND (l.version = $version OR $version IS NULL OR l.version IS NULL)
			RETURN COUNT(l) > 0 AS exists
			`,
			map[string]any{
				"name":    lib.Name,
				"version": lib.Version,
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
			"MERGE (l:Library {name: $name, version: $version})",
			map[string]any{
				"name":    lib.Name,
				"version": lib.Version,
			},
		)
	})
	return multierr.Append(err, session.Close(ctx))
}

func bulkUpsertCallGraphDependencies(ctx context.Context, man *neo4jSvc.Manager, lib *library) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}

	// Insert package-by-package to reduce bandwidth impact
	var merr error
	for _, p := range lib.Packages {
		_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			return tx.Run(ctx,
				`
				UNWIND $callGraphDependencies AS cgd
					// Match the "from" and "to" Symbol nodes by their identity
					MATCH (from:Symbol {identity: cgd.from})
					MATCH (to:Symbol {identity: cgd.to})

					// Check if an CallGraphDependency already exists for the given "from" Symbol
					OPTIONAL MATCH (existingDep:CallGraphDependency)-[:CALLER]->(from)

					// If no existing CallGraphDependency, create one and link it to both "from" and "to"
					FOREACH (_ IN CASE WHEN existingDep IS NULL THEN [1] ELSE [] END |
						CREATE (a:CallGraphDependency)
						CREATE (a)-[:CALLER]->(from)
						CREATE (a)-[:CALLEES]->(to)
					)

					// If an existing CallGraphDependency exists, ensure it is connected to the "to" Symbol
					FOREACH (_ IN CASE WHEN existingDep IS NOT NULL THEN [1] ELSE [] END |
						MERGE (existingDep)-[:CALLEES]->(to)
					)
				`,
				map[string]any{
					"callGraphDependencies": func() []map[string]string {
						out := []map[string]string{}
						// For all symbols from this package
						for _, s := range p.Symbols {
							// For all packages of callees
							for _, callees := range s.Dependencies {
								// For all callees
								for callee := range callees {
									out = append(out, map[string]string{
										"from": s.Identity,
										"to":   callee,
									})
								}
							}
						}
						return out
					}(),
				},
			)
		})
		merr = multierr.Append(merr, err)
	}
	return multierr.Append(merr, session.Close(ctx))
}

func retrieveLibrary(ctx context.Context, man *neo4jSvc.Manager, name, version string) (*Library, error) {
	session, err := man.NewSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx,
			`
			MATCH (l:Library {name: $name, version: $version})
			OPTIONAL MATCH (l)-[:PROVIDES]->(s:Symbol)
			RETURN l, collect(s) AS symbols
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
			return nil, errors.New("library not found")
		}
		rec := res.Record()

		libNode := rec.Values[0].(neo4j.Node)
		symsItf := rec.Values[1].([]any)
		syms := make([]*Symbol, 0, len(symsItf))
		for _, n := range symsItf {
			node := n.(neo4j.Node)
			syms = append(syms, &Symbol{
				Identity: node.Props["identity"].(string),
			})
		}

		return &Library{
			Name:    libNode.Props["name"].(string),
			Version: libNode.Props["version"].(string),
			Provide: syms,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return res.(*Library), nil
}

func retrieveSymbolCallGraphDependencies(ctx context.Context, man *neo4jSvc.Manager, sym *Symbol) (*SymbolDepGraph, error) {
	session, err := man.NewSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx,
			`
			MATCH (from:Symbol {identity: $identity})
			OPTIONAL MATCH (from:Symbol)<-[:FROM]-(:CallGraphDependency)-[:TO]->(s:Symbol)
			RETURN from, collect(s) AS to
			`,
			map[string]any{
				"identity": sym.GetIdentity(),
			},
		)
		if err != nil {
			return nil, err
		}
		if !res.Next(ctx) {
			return nil, errors.New("symbol not found")
		}
		rec := res.Record()

		symNode := rec.Values[0].(neo4j.Node)
		tosItf := rec.Values[1].([]any)
		tos := make([]*Symbol, 0, len(tosItf))
		for _, n := range tosItf {
			node := n.(neo4j.Node)
			tos = append(tos, &Symbol{
				Identity: node.Props["identity"].(string),
			})
		}

		return &SymbolDepGraph{
			From: &Symbol{
				Identity: symNode.Props["identity"].(string),
			},
			To: tos,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return res.(*SymbolDepGraph), nil
}

func reset(ctx context.Context, man *neo4jSvc.Manager) error {
	return multierr.Combine(
		common.Trash(ctx, man, `MATCH (:Library)-[r:PROVIDES]->(:Symbol)`, "r"),
		common.Trash(ctx, man, `MATCH (:Symbol)<-[r:CALLER]-(:CallGraphDependency)`, "r"),
		common.Trash(ctx, man, `MATCH (:Symbol)<-[r:CALLEES]-(:CallGraphDependency)`, "r"),
		common.Trash(ctx, man, `MATCH (n) WHERE n:Library OR n:Symbol OR n:CallGraphDependency`, "n"),
	)
}
