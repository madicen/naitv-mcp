# review-edit-before-approve.tape — press e on a proposal to open the edit form,
# tweak the body, save, and watch it land in the active entries list.
#
# Regen with: vhs vhs/review-edit-before-approve.tape
# Output:     screenshots/review-edit-before-approve.gif

Output screenshots/review-edit-before-approve.gif

Set FontSize 14
Set Width 1300
Set Height 700
Set Theme "Builtin Dark"
Set TypingSpeed 65ms
Set PlaybackSpeed 1.0

Hide
Type "NAITV_MCP_DEMO_DIR=tmp/demo ./naitv-mcp --demo"
Enter
Sleep 1.5s
Show

# Entries tab with badge showing 2 pending proposals.
Sleep 1.5s

# Switch to Review tab.
Tab
Sleep 800ms

# naitv-mcp proposal is selected.
Sleep 1.5s

# Press 'e' to open edit form before approving.
Type "e"
Sleep 800ms

# Form opens pre-populated with proposal values. Tab to body.
Tab
Tab
Sleep 400ms

# Append extra context (single-line field; cursor is already at the end).
Type " Written in Go. See internal/mcp/server.go for tool definitions."
Sleep 500ms

# Save & approve with ctrl+s.
Ctrl+S
Sleep 1.2s

# Status: "Proposal approved after edit."
Sleep 1.5s

# Switch to Entries tab — new entry appears.
Tab
Sleep 1s
Sleep 2s

Hide
Type "q"
Sleep 500ms
