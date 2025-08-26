package apiv1rdg_test

import (
	"bytes"
	_ "embed"
	"testing"

	apiv1rdg "github.com/pandatix/godepgraph/api/v1/rdg"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed state.json
	state []byte
)

func Test_U_AnalyzerProcess(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		In        []byte
		ExpectErr bool
	}{
		"state": {
			In:        state,
			ExpectErr: false,
		},
	}

	for testname, tt := range tests {
		t.Run(testname, func(t *testing.T) {
			a := apiv1rdg.NewAnalyser(nil)
			ana, err := a.Process(t.Context(), bytes.NewReader(tt.In))
			if tt.ExpectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			t.Log(ana.Mermaid())
		})
	}
}
