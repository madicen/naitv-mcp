package review

import (
	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// LoadProposalsCmd loads all pending proposals from the store.
func LoadProposalsCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		proposals, _ := st.ListPending()
		return ProposalsLoadedMsg{Proposals: proposals}
	}
}

// ApproveCmd approves a pending proposal.
func ApproveCmd(st *store.Store, id string) tea.Cmd {
	return func() tea.Msg {
		e, _ := st.Approve(id)
		return ProposalApprovedMsg{Entry: e}
	}
}

// RejectCmd rejects a pending proposal.
func RejectCmd(st *store.Store, id string) tea.Cmd {
	return func() tea.Msg {
		_ = st.Reject(id)
		return ProposalRejectedMsg{ID: id}
	}
}

// ApproveAllCmd approves all pending proposals.
func ApproveAllCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		entries, _ := st.ApproveAll()
		return AllApprovedMsg{Entries: entries}
	}
}

// LoadTargetsCmd loads active entries referenced by update proposals.
func LoadTargetsCmd(st *store.Store, proposals []entry.Entry) tea.Cmd {
	return func() tea.Msg {
		targets := make(map[string]entry.Entry)
		for _, p := range proposals {
			if p.TargetID == "" {
				continue
			}
			if _, ok := targets[p.TargetID]; ok {
				continue
			}
			if e, err := st.Get(p.TargetID); err == nil {
				targets[p.TargetID] = e
			}
		}
		return TargetsLoadedMsg{Targets: targets}
	}
}
