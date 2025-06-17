package cdn

import "encoding/json"

type (
	Workspace struct {
		GoVersion *string `json:"go_version,omitempty"`

		Modules []*Module `json:"modules"` // 1..*
	}

	Module struct {
		Name    string `json:"name"`
		Version string `json:"version"`

		Packages []*Package `json:"packages"` // 1..*
	}

	Package struct {
		Name string `json:"name"`

		Functions Functions `json:"functions"` // 1..*
	}

	Function struct {
		Identity string

		Package *Package

		// Dependencies is a map of packages and function this
		// one depends upon.
		Dependencies map[string]map[string]struct{} // 0..*
	}

	Functions map[string]*Function
)

var _ json.Marshaler = Functions{}

func (fs Functions) MarshalJSON() ([]byte, error) {
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
