package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

type entryProposalSpec struct {
	TargetID   string
	Kind       string
	Name       string
	Group      string
	Body       string
	Tags       []string
	Fields     map[string]string
	ProposedBy string
}

type queuedProposalResult struct {
	Status     string `json:"status"`
	ProposalID string `json:"proposal_id"`
	Message    string `json:"message"`
}

func proposeEntry(st *store.Store, spec entryProposalSpec) (queuedProposalResult, error) {
	proposal := entry.Entry{
		TargetID:   spec.TargetID,
		Kind:       spec.Kind,
		Name:       spec.Name,
		Group:      spec.Group,
		Body:       spec.Body,
		Tags:       spec.Tags,
		Fields:     spec.Fields,
		ProposedBy: spec.ProposedBy,
	}

	created, err := st.CreatePending(proposal)
	if err != nil {
		return queuedProposalResult{}, err
	}

	return queuedProposalResult{
		Status:     "queued",
		ProposalID: created.ID,
		Message:    "Queued for review in naitv-mcp TUI. Run 'naitv-mcp tui' to approve.",
	}, nil
}

func marshalProposalResult(result queuedProposalResult) (string, error) {
	b, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal proposal result: %w", err)
	}
	return string(b), nil
}
