package sig

import "time"

type (
	Component struct {
		Name         string
		Interactions []*Interaction // 1..*
	}

	Interaction struct {
		Timestamp time.Time
		To        string
		Name      string
	}
)
