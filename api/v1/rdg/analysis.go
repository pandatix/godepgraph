package apiv1rdg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

var DefaultAnalyserParams = &AnalyserParams{}

type Analyser struct {
	*AnalyserParams
}

type AnalyserParams struct{}

type Analysis struct {
	Assets   []*Resource
	Pairings []Pairing
}

func NewAnalyser(params *AnalyserParams) *Analyser {
	if params == nil {
		params = DefaultAnalyserParams
	}
	return &Analyser{
		AnalyserParams: params,
	}
}

func (a *Analyser) Process(ctx context.Context, r io.Reader) (*Analysis, error) {
	var udep apitype.UntypedDeployment
	if err := json.NewDecoder(r).Decode(&udep); err != nil {
		return nil, err
	}

	switch udep.Version {
	case 3:
		var dep apitype.DeploymentV3
		if err := json.Unmarshal(udep.Deployment, &dep); err != nil {
			return nil, err
		}

		ana := &Analysis{
			Assets:   []*Resource{},
			Pairings: []Pairing{}, // cannot determine this, in [0;n^n]
		}

		for _, r := range dep.Resources {
			// Look for an existing resource transformer. If none known, skip.
			p := strings.Split(r.Type.String(), ":")[0]
			t, ok := GetRegistry().Load(p)
			if !ok {
				continue
			}

			// If the transformer doesn't care about this resource, skip.
			res := t(r)
			if res == nil {
				continue
			}
			ana.Assets = append(ana.Assets, res)

			// TODO extract pairings
		}

		return ana, nil

	default:
		return nil, fmt.Errorf("unsupport version %d", udep.Version)
	}
}

func (ana *Analysis) Mermaid() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "flowchart TD\n")
	for i, r := range ana.Assets {
		fmt.Fprintf(sb, "\tR%d[%s]\n", i, r.ID)
	}
	for _, p := range ana.Pairings {
		var from, to int
		for i, r := range ana.Assets {
			if p.FromRes == r.ID {
				from = i
			}
			if p.ToRes == r.ID {
				to = i
			}
		}
		fmt.Fprintf(sb, "\tR%d --> R%d\n", from, to)
	}
	return sb.String()
}
