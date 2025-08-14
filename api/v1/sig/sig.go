package apiv1sig

func NewSIG() *SIG {
	return &SIG{}
}

// SIG service.
type SIG struct {
	UnimplementedSIGServer
}
