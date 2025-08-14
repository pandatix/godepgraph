package apiv1cdn

func NewCDN() *CDN {
	return &CDN{}
}

// CDN service.
type CDN struct {
	UnimplementedCDNServer
}
