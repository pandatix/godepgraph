package sig

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var DefaultAnalyserParams = &AnalyserParams{}

type Analyser struct {
	*AnalyserParams
}

type AnalyserParams struct{}

type Analysis struct {
	components map[string]*Component
}

func NewAnalyser(params *AnalyserParams) *Analyser {
	if params == nil {
		params = DefaultAnalyserParams
	}
	return &Analyser{
		AnalyserParams: params,
	}
}

func (a *Analyser) Process(f io.Reader) (*Analysis, error) {
	// Get all spans per traces and service names
	traces, err := a.parseTraces(f)
	if err != nil {
		return nil, err
	}

	// Run the analysis
	ana := &Analysis{
		components: map[string]*Component{},
	}
	for _, comp := range traces {
		for compName, spans := range comp {
			for _, span := range spans {
				// Register this component if not already
				if _, ok := ana.components[compName]; !ok {
					ana.components[compName] = &Component{
						Name:         compName,
						Interactions: []*Interaction{},
					}
				}

				pid := span.ParentSpanID()
				if pid.IsEmpty() {
					continue
				}
				pname := getSpanParentName(comp, pid)
				if pname == "" {
					// XXX this case should be covered but need larger control over ALL spans at once
					continue
				}

				// For every upstream interaction, register it not already
				pcomp, ok := ana.components[pname]
				if !ok {
					pcomp = &Component{
						Name:         pname,
						Interactions: []*Interaction{},
					}
					ana.components[pname] = pcomp
				}

				// Then register the interaction
				pcomp.Interactions = append(pcomp.Interactions, &Interaction{
					Timestamp: span.StartTimestamp().AsTime(),
					To:        compName,
					Name:      span.Name(),
				})
			}
		}
	}
	return ana, nil
}

func getSpanParentName(mp map[string][]ptrace.Span, id pcommon.SpanID) string {
	hx := hex.EncodeToString(id[:])
	for compName, spans := range mp {
		for _, span := range spans {
			spid := span.SpanID()
			sx := hex.EncodeToString(spid[:])
			if hx == sx {
				return compName
			}
		}
	}
	return ""
}

func (a *Analyser) parseTraces(f io.Reader) (map[string]map[string][]ptrace.Span, error) {
	tracesUnmarshaler := &ptrace.JSONUnmarshaler{}
	spans := map[string]map[string][]ptrace.Span{} // per traceID, per service.name, spans
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		t, err := tracesUnmarshaler.UnmarshalTraces(scan.Bytes())
		if err != nil {
			return nil, err
		}

		if t.ResourceSpans().Len() != 1 {
			return nil, fmt.Errorf("multiple resource spans %v", t)
		}

		rs := t.ResourceSpans().At(0)
		sn, _ := rs.Resource().Attributes().Get("service.name")
		for i := 0; i < rs.ScopeSpans().Len(); i++ {
			s := rs.ScopeSpans().At(0)
			for j := 0; j < s.Spans().Len(); j++ {
				sp := s.Spans().At(j)
				traceID := sp.TraceID().String()

				if _, ok := spans[traceID]; !ok {
					spans[traceID] = map[string][]ptrace.Span{}
				}
				if _, ok := spans[traceID][sn.Str()]; !ok {
					spans[traceID][sn.Str()] = []ptrace.Span{}
				}
				spans[traceID][sn.Str()] = append(spans[traceID][sn.Str()], sp)
			}
		}
	}
	return spans, nil
}
