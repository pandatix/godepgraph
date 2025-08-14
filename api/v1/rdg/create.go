package apiv1rdg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"go.opentelemetry.io/otel/trace"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
)

func (*RDG) CreateStack(ctx context.Context, req *CreateStackRequest) (*Stack, error) {
	logger := global.Log()
	span := trace.SpanFromContext(ctx)

	logger.Info(ctx, "creating stack")

	// Download state
	span.AddEvent("downloading state")
	r, err := download(req.GetUri())
	if err != nil {
		return nil, err
	}
	if r, ok := r.(io.Closer); ok {
		defer r.Close()
	}

	// Parse state
	span.AddEvent("processing state")
	ana := NewAnalyser(&AnalyserParams{})
	res, err := ana.Process(ctx, r)
	if err != nil {
		return nil, err
	}

	span.AddEvent("processed state, exporting it")
	if err := res.Export(ctx, global.GetNeo4JManager()); err != nil {
		return nil, err
	}

	return &Stack{}, nil
}

func download(uri string) (io.Reader, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		r, err := http.Get(uri)
		if err != nil {
			return nil, err
		}
		return r.Body, nil

	case "file":
		return os.Open(u.Path)

	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}
