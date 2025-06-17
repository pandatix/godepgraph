package rdg

type (
	Resource struct {
		ID string
	}

	Pairing struct {
		FromRes string
		FromOut string

		ToRes string
		ToIn  string
	}
)
