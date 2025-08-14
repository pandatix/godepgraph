package common

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	neo4jSvc "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/pkg/services/neo4j"
)

const (
	limit int = 10_000
)

func Trash(ctx context.Context, man *neo4jSvc.Manager, cypher, n string) error {
	return paginated(ctx, man,
		fmt.Sprintf(`%[1]s WITH %[2]s LIMIT %[3]d DELETE %[2]s RETURN count(*) AS deleted`, cypher, n, limit),
		nil,
	)
}

// pagniated executes a query iteratively until there is no remaining work to do.
func paginated(ctx context.Context, man *neo4jSvc.Manager, cypher string, params map[string]any) error {
	session, err := man.NewSession(ctx)
	if err != nil {
		return err
	}
	for {
		rem, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			result, err := tx.Run(ctx, cypher, params)
			if err != nil {
				return nil, err
			}
			if result.Next(ctx) {
				return result.Record().Values[0].(int64), nil
			}
			return int64(0), result.Err()
		})
		if err != nil {
			return err
		}
		if rem.(int64) == 0 {
			break
		}
	}
	return nil
}
