package apiv1rdg

func NewRDG() *RDG {
	return &RDG{}
}

// RDG service.
type RDG struct {
	UnimplementedRDGServer
}
