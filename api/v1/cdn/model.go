package apiv1cdn

import "encoding/json"

type (
	Workspace struct {
		GoVersion *string `json:"go_version,omitempty"`

		Modules []*library `json:"modules"` // 1..*
	}

	library struct {
		Name    string `json:"name"`
		Version string `json:"version"`

		Packages []*pkg `json:"packages"` // 1..*
	}

	pkg struct {
		Name string `json:"name"`

		Symbols Symbols `json:"symbols"` // 1..*
	}

	symbol struct {
		Identity string

		Package *pkg

		// Dependencies is a map of packages and function this
		// one depends upon.
		Dependencies map[string]map[string]struct{} // 0..*
	}

	Symbols map[string]*symbol
)

var _ json.Marshaler = Symbols{}

func (fs Symbols) MarshalJSON() ([]byte, error) {
	out := make([]map[string]any, 0, len(fs))
	for id, f := range fs {
		deps := []string{}
		for _, pdep := range f.Dependencies {
			for dep := range pdep {
				deps = append(deps, dep)
			}
		}

		out = append(out, map[string]any{
			"identity":     id,
			"dependencies": deps,
		})
	}
	return json.Marshal(out)
}
