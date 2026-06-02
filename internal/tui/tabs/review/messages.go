package review

import "github.com/madicen/naitv-mcp/pkg/entry"

// ProposalsLoadedMsg is sent when proposals have been loaded from the store.
type ProposalsLoadedMsg struct {
	Proposals []entry.Entry
}

// ProposalApprovedMsg is sent when a proposal has been approved.
type ProposalApprovedMsg struct {
	Entry entry.Entry
}

// ProposalRejectedMsg is sent when a proposal has been rejected.
type ProposalRejectedMsg struct {
	ID string
}

// AllApprovedMsg is sent when all proposals have been approved.
type AllApprovedMsg struct {
	Entries []entry.Entry
}
