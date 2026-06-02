package review

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/naitv-mcp/internal/store"
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
		st.Reject(id)
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
