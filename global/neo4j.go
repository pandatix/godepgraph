package global

import (
	"context"
	"sync"

	"github.com/sony/gobreaker/v2"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/pkg/services/neo4j"
)

var (
	neo4jInstance *neo4j.Manager
	neo4jOnce     sync.Once
)

func GetNeo4JManager() *neo4j.Manager {
	neo4jOnce.Do(func() {
		neo4jInstance = neo4j.NewManager(neo4j.Config{
			URL:      Conf.Neo4J.URL,
			Username: Conf.Neo4J.User,
			Password: Conf.Neo4J.Pass,
			DBName:   Conf.Neo4J.DBName,
			CBOnStateChange: func(name string, from, to gobreaker.State) {
				Log().Info(context.Background(), "circuit breaker state change",
					zap.String("circuit", name),
					zap.String("from", from.String()),
					zap.String("to", to.String()),
				)
			},
		})
	})
	return neo4jInstance
}

type Neo4JInitializer func(ctx context.Context, d *neo4j.Manager) error

var (
	neo4jInits   []Neo4JInitializer
	neo4jInitsMx sync.Mutex
)

func RegisterNeo4JInitializer(i Neo4JInitializer) {
	neo4jInitsMx.Lock()
	defer neo4jInitsMx.Unlock()

	neo4jInits = append(neo4jInits, i)
}

func ExecuteNeo4JInitializers(ctx context.Context, d *neo4j.Manager) (err error) {
	neo4jInitsMx.Lock()
	defer neo4jInitsMx.Unlock()

	for _, i := range neo4jInits {
		err = multierr.Append(err, i(ctx, d))
	}
	return
}
