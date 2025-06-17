package rdg

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
	Resources []*Resource
	Pairings  []Pairing
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
			Resources: make([]*Resource, len(dep.Resources)),
			Pairings:  []Pairing{}, // cannot determine this, best is 0, worst is n^n
		}

		for i, r := range dep.Resources {
			ana.Resources[i] = &Resource{
				ID: string(r.URN),
			}
			for pk, urns := range r.PropertyDependencies {
				for _, urn := range urns {
					ana.Pairings = append(ana.Pairings, Pairing{
						FromRes: string(urn),
						FromOut: "?",
						ToRes:   string(r.URN),
						ToIn:    string(pk),
					})
				}
			}
		}

		return ana, nil

	default:
		return nil, fmt.Errorf("unsupport version %d", udep.Version)
	}
}

func (ana *Analysis) Mermaid() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "flowchart TD\n")
	for i, r := range ana.Resources {
		fmt.Fprintf(sb, "\tR%d[%s]\n", i, r.ID)
	}
	for _, p := range ana.Pairings {
		var from, to int
		for i, r := range ana.Resources {
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
