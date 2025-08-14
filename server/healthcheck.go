package server

import (
	"context"
	"net/http"
	"time"

	"github.com/hellofresh/health-go/v5"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
)

func healthcheck() http.Handler {
	opts := []health.Option{
		health.WithComponent(health.Component{
			Name:    "godepgraph",
			Version: global.Version,
		}),
		health.WithSystemInfo(),
	}
	h, err := health.New(opts...)
	if err != nil {
		panic(err)
	}

	_ = h.Register(health.Config{
		Name:    "neo4j",
		Timeout: time.Second,
		Check: func(ctx context.Context) error {
			man := global.GetNeo4JManager()
			return man.Healthcheck(ctx)
		},
	})

	return h.Handler()
}
