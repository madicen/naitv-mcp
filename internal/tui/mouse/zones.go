package mouse

import "fmt"

const (
	ZoneTabEntries = "tab:entries"
	ZoneTabReview  = "tab:review"

	ZoneKindTab = "kind:" // + kind string

	ZoneEntryRow = "entry:" // + index

	ZoneActionNew    = "action:new"
	ZoneActionEdit   = "action:edit"
	ZoneActionDelete = "action:delete"
	ZoneActionSearch = "action:search"
	ZoneActionReview = "action:review"

	ZoneProposalRow = "proposal:" // + index

	ZoneActionApprove    = "action:approve"
	ZoneActionReject     = "action:reject"
	ZoneActionEditReview = "action:edit-review"
	ZoneActionApproveAll = "action:approve-all"

	ZoneDetailApprove = "detail:approve"
	ZoneDetailReject  = "detail:reject"
	ZoneDetailEdit    = "detail:edit"

	ZoneFormSave        = "form:save"
	ZoneFormCancel      = "form:cancel"
	ZoneFormAddField    = "form:add-field"
	ZoneFormRemoveField = "form:remove-field:" // + index
)

func EntryRowZone(i int) string     { return fmt.Sprintf("%s%d", ZoneEntryRow, i) }
func ProposalRowZone(i int) string  { return fmt.Sprintf("%s%d", ZoneProposalRow, i) }
func KindTabZone(kind string) string { return ZoneKindTab + kind }
func RemoveFieldZone(i int) string  { return fmt.Sprintf("%s%d", ZoneFormRemoveField, i) }
