package entry

import "time"

type Status string

const (
	StatusActive  Status = "active"
	StatusPending Status = "pending"
)

// Delivery controls how an entry is surfaced to agents.
type Delivery string

const (
	// DeliveryInit entries are included in the initialization bundle that
	// agents receive up front (via the initialize tool or `init` command).
	DeliveryInit Delivery = "init"
	// DeliveryOnDemand entries are kept out of the initialization bundle; an
	// agent must ask for them directly via get_entry/search_entries.
	DeliveryOnDemand Delivery = "on-demand"
)

type Entry struct {
	ID         string
	Kind       string
	Name       string
	Body       string
	Tags       []string
	Fields     map[string]string
	Status     Status
	Delivery   Delivery   // how the entry is surfaced to agents (default: init)
	ProposedBy string     // agent name, empty for TUI-created
	ProposedAt *time.Time // non-nil when Status == pending
	TargetID   string     // for update proposals: ID of active entry being modified
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// DeliveryOrDefault returns the entry's delivery mode, defaulting to
// DeliveryInit when unset.
func (e Entry) DeliveryOrDefault() Delivery {
	if e.Delivery == "" {
		return DeliveryInit
	}
	return e.Delivery
}
